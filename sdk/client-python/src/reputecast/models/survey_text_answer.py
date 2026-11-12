from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SurveyTextAnswer")



@_attrs_define
class SurveyTextAnswer:
    """ 
        Attributes:
            at (datetime.datetime):
            text (str):
            contact (str | Unset):
     """

    at: datetime.datetime
    text: str
    contact: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        text = self.text

        contact = self.contact


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "text": text,
        })
        if contact is not UNSET:
            field_dict["contact"] = contact

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        text = d.pop("text")

        contact = d.pop("contact", UNSET)

        survey_text_answer = cls(
            at=at,
            text=text,
            contact=contact,
        )

        return survey_text_answer

