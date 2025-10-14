"""
RFE Workflow Types and Data Structures

Based on vTeam RFE workflow system, adapted for Kubeflow Pipelines.
"""

from typing import List, Optional, Dict, Any
from dataclasses import dataclass
from enum import Enum
import json
from datetime import datetime


class RFEPhase(Enum):
    """RFE workflow phases"""
    INITIALIZING = "Initializing"
    READY = "Ready"
    AGENT_ANALYSIS = "Agent_Analysis"
    SPECIFICATION = "Specification"
    PLANNING = "Planning"
    TASKS_GENERATION = "Tasks_Generation"
    APPROVED = "Approved"
    REJECTED = "Rejected"
    COMPLETED = "Completed"


class AgentRole(Enum):
    """Agent roles in the RFE council"""
    PRODUCT_MANAGER = "Parker"  # PM
    ARCHITECT = "Archie"        # Architect
    STAFF_ENGINEER = "Stella"   # Staff Engineer
    UX_LEAD = "Uma"            # UX Team Lead
    TEAM_LEAD = "Lee"          # Team Lead
    TEST_ENGINEER = "Neil"     # Test Engineer
    DELIVERY_OWNER = "Jack"    # Delivery Owner


@dataclass
class GitRepository:
    """Git repository configuration"""
    url: str
    branch: Optional[str] = "main"
    path: Optional[str] = None

    def to_dict(self) -> Dict[str, Any]:
        return {
            "url": self.url,
            "branch": self.branch,
            "path": self.path
        }


@dataclass
class JiraLink:
    """Link between workspace file and Jira issue"""
    path: str
    jira_key: str

    def to_dict(self) -> Dict[str, Any]:
        return {
            "path": self.path,
            "jiraKey": self.jira_key
        }


@dataclass
class AgentAnalysis:
    """Analysis result from an agent"""
    agent_role: AgentRole
    analysis: str
    score: Optional[float] = None
    recommendations: Optional[List[str]] = None
    concerns: Optional[List[str]] = None
    timestamp: Optional[datetime] = None

    def to_dict(self) -> Dict[str, Any]:
        return {
            "agentRole": self.agent_role.value,
            "analysis": self.analysis,
            "score": self.score,
            "recommendations": self.recommendations or [],
            "concerns": self.concerns or [],
            "timestamp": self.timestamp.isoformat() if self.timestamp else None
        }


@dataclass
class RFESpec:
    """RFE Specification"""
    title: str
    description: str
    umbrella_repo: GitRepository
    supporting_repos: Optional[List[GitRepository]] = None
    workspace_path: Optional[str] = None
    parent_outcome: Optional[str] = None
    jira_links: Optional[List[JiraLink]] = None

    def to_dict(self) -> Dict[str, Any]:
        return {
            "title": self.title,
            "description": self.description,
            "umbrellaRepo": self.umbrella_repo.to_dict(),
            "supportingRepos": [repo.to_dict() for repo in (self.supporting_repos or [])],
            "workspacePath": self.workspace_path,
            "parentOutcome": self.parent_outcome,
            "jiraLinks": [link.to_dict() for link in (self.jira_links or [])]
        }


@dataclass
class RFEStatus:
    """RFE Status tracking"""
    phase: RFEPhase
    message: Optional[str] = None
    agent_analyses: Optional[List[AgentAnalysis]] = None
    spec_generated: bool = False
    plan_generated: bool = False
    tasks_generated: bool = False

    def to_dict(self) -> Dict[str, Any]:
        return {
            "phase": self.phase.value,
            "message": self.message,
            "agentAnalyses": [analysis.to_dict() for analysis in (self.agent_analyses or [])],
            "specGenerated": self.spec_generated,
            "planGenerated": self.plan_generated,
            "tasksGenerated": self.tasks_generated
        }


@dataclass
class RFEWorkflow:
    """Complete RFE Workflow definition"""
    id: str
    spec: RFESpec
    status: RFEStatus
    created_at: datetime
    updated_at: Optional[datetime] = None
    project: Optional[str] = None

    @classmethod
    def create(cls, title: str, description: str, umbrella_repo_url: str,
               umbrella_repo_branch: str = "main",
               supporting_repos: Optional[List[Dict[str, str]]] = None,
               project: Optional[str] = None) -> 'RFEWorkflow':
        """Create a new RFE workflow"""
        import uuid

        rfe_id = f"rfe-{int(datetime.now().timestamp())}"

        umbrella_repo = GitRepository(url=umbrella_repo_url, branch=umbrella_repo_branch)

        supporting_repo_objects = []
        if supporting_repos:
            supporting_repo_objects = [
                GitRepository(url=repo["url"], branch=repo.get("branch", "main"))
                for repo in supporting_repos
            ]

        spec = RFESpec(
            title=title,
            description=description,
            umbrella_repo=umbrella_repo,
            supporting_repos=supporting_repo_objects
        )

        status = RFEStatus(phase=RFEPhase.INITIALIZING)

        return cls(
            id=rfe_id,
            spec=spec,
            status=status,
            created_at=datetime.now(),
            project=project
        )

    def to_dict(self) -> Dict[str, Any]:
        return {
            "id": self.id,
            "spec": self.spec.to_dict(),
            "status": self.status.to_dict(),
            "createdAt": self.created_at.isoformat(),
            "updatedAt": self.updated_at.isoformat() if self.updated_at else None,
            "project": self.project
        }

    def to_kubernetes_crd(self) -> Dict[str, Any]:
        """Convert to Kubernetes Custom Resource format"""
        return {
            "apiVersion": "kfp.kubeflow.org/v1alpha1",
            "kind": "RFEWorkflow",
            "metadata": {
                "name": self.id,
                "namespace": self.project or "default",
                "labels": {
                    "rfe.kfp.kubeflow.org/workflow": self.id,
                    "rfe.kfp.kubeflow.org/phase": self.status.phase.value.lower()
                }
            },
            "spec": self.spec.to_dict(),
            "status": self.status.to_dict()
        }


class KFPComponentType(Enum):
    """KFP-specific component types for RFE generation"""
    PYTHON_COMPONENT = "python_component"
    CONTAINER_COMPONENT = "container_component"
    PIPELINE = "pipeline"
    NOTEBOOK_COMPONENT = "notebook_component"
    IMPORTER = "importer"


@dataclass
class KFPRequirement:
    """KFP-specific requirement"""
    component_type: KFPComponentType
    name: str
    description: str
    inputs: List[str]
    outputs: List[str]
    dependencies: Optional[List[str]] = None

    def to_dict(self) -> Dict[str, Any]:
        return {
            "componentType": self.component_type.value,
            "name": self.name,
            "description": self.description,
            "inputs": self.inputs,
            "outputs": self.outputs,
            "dependencies": self.dependencies or []
        }