from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="SurveyProblem")



@_attrs_define
class SurveyProblem:
    """ 
        Attributes:
            field (str):
            message (str):
            question (str | Unset):
     """

    field: str
    message: str
    question: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        field = self.field

        message = self.message

        question = self.question


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "field": field,
            "message": message,
        })
        if question is not UNSET:
            field_dict["question"] = question

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        field = d.pop("field")

        message = d.pop("message")

        question = d.pop("question", UNSET)

        survey_problem = cls(
            field=field,
            message=message,
            question=question,
        )

        return survey_problem

