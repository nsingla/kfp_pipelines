import json

from pydantic import BaseModel

class JsonDeserializationUtils:
    """
    Helper class to deserialize the json file.
    """

    @staticmethod
    def get_list_from_file(file_path: str, data_model):
        """
        Helper method to convert the json file into list of TestScenario objects
        :param file_path: json file path
        :param data_model: data model to convert
        :return: list of objects of type data model.
        """
        with open(file_path) as input_file:
            list_from_json = json.load(input_file)
            return [data_model(**json_string) for json_string in list_from_json]

    @staticmethod
    def get_list_from_dictionary(json_data: dict, data_model: BaseModel):
        """
        Helper method to convert the json file into list of TestScenario objects
        :param json_data: dictionary of the json data
        :param data_model: data model to convert to
        :return: list of objects of type data model.
        """
        return [data_model(**json_string) for json_string in json_data]