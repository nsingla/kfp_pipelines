// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/golang/glog"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

const artifactTaskTableName = "artifact_tasks"

var artifactTaskColumns = []string{
	"artifact_tasks.UUID",
	"artifact_tasks.ArtifactID",
	"artifact_tasks.TaskID",
	"artifact_tasks.Type",
	"artifact_tasks.RunUUID",
	"artifact_tasks.ProducerTaskName",
	"artifact_tasks.ProducerKey",
	"artifact_tasks.ArtifactKey",
}

type ArtifactTaskStoreInterface interface {
	// CreateArtifactTask Create an artifact-task relationship entry in the database.
	CreateArtifactTask(artifactTask *model.ArtifactTask) (*model.ArtifactTask, error)

	// CreateArtifactTasks Create multiple artifact-task relationships in a single transaction.
	CreateArtifactTasks(artifactTasks []*model.ArtifactTask) ([]*model.ArtifactTask, error)

	// GetArtifactTask Fetches an artifact-task relationship with a given id.
	GetArtifactTask(id string) (*model.ArtifactTask, error)

	// ListArtifactTasks Fetches artifact-task relationships for given filtering and listing options.
	// filterContexts supports multiple filters: ArtifactID, TaskID, RunUUID, or Type
	ListArtifactTasks(filterContexts []*model.FilterContext, opts *list.Options) ([]*model.ArtifactTask, int, string, error)
}

type ArtifactTaskStore struct {
	db   *DB
	uuid util.UUIDGeneratorInterface
}

// NewArtifactTaskStore creates a new ArtifactTaskStore.
func NewArtifactTaskStore(db *DB, uuid util.UUIDGeneratorInterface) *ArtifactTaskStore {
	return &ArtifactTaskStore{
		db:   db,
		uuid: uuid,
	}
}

func (s *ArtifactTaskStore) CreateArtifactTask(artifactTask *model.ArtifactTask) (*model.ArtifactTask, error) {
	// Set up UUID for artifact-task relationship.
	newArtifactTask := *artifactTask
	id, err := s.uuid.NewRandom()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create an artifact-task id")
	}
	newArtifactTask.UUID = id.String()

	sql, args, err := sq.
		Insert(artifactTaskTableName).
		SetMap(
			sq.Eq{
				"UUID":             newArtifactTask.UUID,
				"ArtifactID":       newArtifactTask.ArtifactID,
				"TaskID":           newArtifactTask.TaskID,
				"Type":             newArtifactTask.Type,
				"RunUUID":          newArtifactTask.RunUUID,
				"ProducerTaskName": newArtifactTask.ProducerTaskName,
				"ProducerKey":      newArtifactTask.ProducerKey,
				"ArtifactKey":      newArtifactTask.ArtifactKey,
			},
		).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to insert artifact-task to artifact_tasks table: %v",
			err.Error())
	}

	_, err = s.db.Exec(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to add artifact-task to artifact_tasks table: %v",
			err.Error())
	}

	return &newArtifactTask, nil
}

func (s *ArtifactTaskStore) CreateArtifactTasks(artifactTasks []*model.ArtifactTask) ([]*model.ArtifactTask, error) {
	if len(artifactTasks) == 0 {
		return []*model.ArtifactTask{}, nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to start transaction for creating artifact-tasks")
	}
	defer tx.Rollback()

	var newArtifactTasks []*model.ArtifactTask
	for _, artifactTask := range artifactTasks {
		newArtifactTask := *artifactTask
		id, err := s.uuid.NewRandom()
		if err != nil {
			return nil, util.NewInternalServerError(err, "Failed to create an artifact-task id")
		}
		newArtifactTask.UUID = id.String()

		sql, args, err := sq.
			Insert(artifactTaskTableName).
			SetMap(
				sq.Eq{
					"UUID":             newArtifactTask.UUID,
					"ArtifactID":       newArtifactTask.ArtifactID,
					"TaskID":           newArtifactTask.TaskID,
					"Type":             newArtifactTask.Type,
					"RunUUID":          newArtifactTask.RunUUID,
					"ProducerTaskName": newArtifactTask.ProducerTaskName,
					"ProducerKey":      newArtifactTask.ProducerKey,
					"ArtifactKey":      newArtifactTask.ArtifactKey,
				},
			).
			ToSql()
		if err != nil {
			return nil, util.NewInternalServerError(err, "Failed to create query to insert artifact-task: %v", err.Error())
		}

		_, err = tx.Exec(sql, args...)
		if err != nil {
			return nil, util.NewInternalServerError(err, "Failed to add artifact-task: %v", err.Error())
		}

		newArtifactTasks = append(newArtifactTasks, &newArtifactTask)
	}

	err = tx.Commit()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to commit transaction for creating artifact-tasks")
	}

	return newArtifactTasks, nil
}

func (s *ArtifactTaskStore) scanRows(rows *sql.Rows) ([]*model.ArtifactTask, error) {
	var artifactTasks []*model.ArtifactTask
	for rows.Next() {
		var uuid, artifactID, taskID string
		var runUUID, producerTaskName, producerKey, artifactKey string
		var artifactTaskType int32

		err := rows.Scan(
			&uuid,
			&artifactID,
			&taskID,
			&artifactTaskType,
			&runUUID,
			&producerTaskName,
			&producerKey,
			&artifactKey,
		)
		if err != nil {
			return artifactTasks, err
		}

		// Convert string type back to enum type
		artifactTask := &model.ArtifactTask{
			UUID:             uuid,
			ArtifactID:       artifactID,
			TaskID:           taskID,
			Type:             apiv2beta1.ArtifactTaskType(artifactTaskType),
			RunUUID:          runUUID,
			ProducerTaskName: producerTaskName,
			ProducerKey:      producerKey,
			ArtifactKey:      artifactKey,
		}
		artifactTasks = append(artifactTasks, artifactTask)
	}
	return artifactTasks, nil
}

// applyFilterContextsToQuery applies multiple filter contexts to the query builder
// Supports filtering by multiple artifact_ids, task_ids, and run_ids simultaneously
func (s *ArtifactTaskStore) applyFilterContextsToQuery(sqlBuilder sq.SelectBuilder, filterContexts []*model.FilterContext) sq.SelectBuilder {
	var artifactIDs []string
	var taskIDs []string
	var runIDs []string

	// Collect all filter values by type
	for _, filterContext := range filterContexts {
		if filterContext == nil || filterContext.ReferenceKey == nil {
			continue
		}

		switch filterContext.ReferenceKey.Type {
		case model.ArtifactResourceType:
			artifactIDs = append(artifactIDs, filterContext.ReferenceKey.ID)
		case model.TaskResourceType:
			taskIDs = append(taskIDs, filterContext.ReferenceKey.ID)
		case model.RunResourceType:
			runIDs = append(runIDs, filterContext.ReferenceKey.ID)
		}
	}

	// Apply artifact ID filters (OR within artifact IDs)
	if len(artifactIDs) > 0 {
		if len(artifactIDs) == 1 {
			sqlBuilder = sqlBuilder.Where(sq.Eq{"artifact_tasks.ArtifactID": artifactIDs[0]})
		} else {
			sqlBuilder = sqlBuilder.Where(sq.Eq{"artifact_tasks.ArtifactID": artifactIDs})
		}
	}

	// Apply task ID filters (OR within task IDs)
	if len(taskIDs) > 0 {
		if len(taskIDs) == 1 {
			sqlBuilder = sqlBuilder.Where(sq.Eq{"artifact_tasks.TaskID": taskIDs[0]})
		} else {
			sqlBuilder = sqlBuilder.Where(sq.Eq{"artifact_tasks.TaskID": taskIDs})
		}
	}

	// Apply run ID filters (OR within run IDs) now directly on artifact_tasks
	if len(runIDs) > 0 {
		if len(runIDs) == 1 {
			sqlBuilder = sqlBuilder.Where(sq.Eq{"artifact_tasks.RunUUID": runIDs[0]})
		} else {
			sqlBuilder = sqlBuilder.Where(sq.Eq{"artifact_tasks.RunUUID": runIDs})
		}
	}

	return sqlBuilder
}

func (s *ArtifactTaskStore) ListArtifactTasks(filterContexts []*model.FilterContext, opts *list.Options) ([]*model.ArtifactTask, int, string, error) {
	errorF := func(err error) ([]*model.ArtifactTask, int, string, error) {
		return nil, 0, "", util.NewInternalServerError(err, "Failed to list artifact-tasks: %v", err)
	}

	// SQL for getting the filtered and paginated rows
	sqlBuilder := sq.Select(artifactTaskColumns...).From(artifactTaskTableName)
	sqlBuilder = s.applyFilterContextsToQuery(sqlBuilder, filterContexts)
	sqlBuilder = opts.AddFilterToSelect(sqlBuilder)

	rowsSql, rowsArgs, err := opts.AddPaginationToSelect(sqlBuilder).ToSql()
	if err != nil {
		return errorF(err)
	}

	// SQL for getting total size
	countBuilder := sq.Select("count(*)").From(artifactTaskTableName)
	countBuilder = s.applyFilterContextsToQuery(countBuilder, filterContexts)
	sizeSql, sizeArgs, err := opts.AddFilterToSelect(countBuilder).ToSql()
	if err != nil {
		return errorF(err)
	}

	// Use a transaction to make sure we're returning the total_size of the same rows queried
	tx, err := s.db.Begin()
	if err != nil {
		glog.Errorf("Failed to start transaction to list artifact-tasks")
		return errorF(err)
	}

	rows, err := tx.Query(rowsSql, rowsArgs...)
	if err != nil {
		tx.Rollback()
		return errorF(err)
	}
	if err := rows.Err(); err != nil {
		tx.Rollback()
		return errorF(err)
	}
	artifactTasks, err := s.scanRows(rows)
	if err != nil {
		tx.Rollback()
		return errorF(err)
	}
	defer rows.Close()

	sizeRow, err := tx.Query(sizeSql, sizeArgs...)
	if err != nil {
		tx.Rollback()
		return errorF(err)
	}
	if err := sizeRow.Err(); err != nil {
		tx.Rollback()
		return errorF(err)
	}
	total_size, err := list.ScanRowToTotalSize(sizeRow)
	if err != nil {
		tx.Rollback()
		return errorF(err)
	}
	defer sizeRow.Close()

	err = tx.Commit()
	if err != nil {
		glog.Errorf("Failed to commit transaction to list artifact-tasks")
		return errorF(err)
	}

	if len(artifactTasks) <= opts.PageSize {
		return artifactTasks, total_size, "", nil
	}

	npt, err := opts.NextPageToken(artifactTasks[opts.PageSize])
	return artifactTasks[:opts.PageSize], total_size, npt, err
}

func (s *ArtifactTaskStore) GetArtifactTask(id string) (*model.ArtifactTask, error) {
	sql, args, err := sq.
		Select(artifactTaskColumns...).
		From(artifactTaskTableName).
		Where(sq.Eq{"UUID": id}).
		Limit(1).ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to get artifact-task: %v", err.Error())
	}

	r, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get artifact-task: %v", err.Error())
	}
	defer r.Close()

	artifactTasks, err := s.scanRows(r)
	if err != nil || len(artifactTasks) > 1 {
		return nil, util.NewInternalServerError(err, "Failed to get artifact-task: %v", err.Error())
	}
	if len(artifactTasks) == 0 {
		return nil, util.NewResourceNotFoundError("artifact-task", fmt.Sprint(id))
	}

	return artifactTasks[0], nil
}
