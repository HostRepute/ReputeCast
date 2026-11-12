from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SurveyResponse")



@_attrs_define
class SurveyResponse:
    """ 
        Attributes:
            answers (list[str]):
            id (str):
            non_contact (bool):
            completed_at (datetime.datetime | Unset):
            contact (str | Unset):
            contact_id (str | Unset):
     """

    answers: list[str]
    id: str
    non_contact: bool
    completed_at: datetime.datetime | Unset = UNSET
    contact: str | Unset = UNSET
    contact_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        answers = self.answers



        id = self.id

        non_contact = self.non_contact

        completed_at: str | Unset = UNSET
        if not isinstance(self.completed_at, Unset):
            completed_at = self.completed_at.isoformat()

        contact = self.contact

        contact_id = self.contact_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "answers": answers,
            "id": id,
            "nonContact": non_contact,
        })
        if completed_at is not UNSET:
            field_dict["completedAt"] = completed_at
        if contact is not UNSET:
            field_dict["contact"] = contact
        if contact_id is not UNSET:
            field_dict["contactId"] = contact_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        answers = cast(list[str], d.pop("answers"))


        id = d.pop("id")

        non_contact = d.pop("nonContact")

        _completed_at = d.pop("completedAt", UNSET)
        completed_at: datetime.datetime | Unset
        if isinstance(_completed_at,  Unset):
            completed_at = UNSET
        else:
            completed_at = datetime.datetime.fromisoformat(_completed_at)




        contact = d.pop("contact", UNSET)

        contact_id = d.pop("contactId", UNSET)

        survey_response = cls(
            answers=answers,
            id=id,
            non_contact=non_contact,
            completed_at=completed_at,
            contact=contact,
            contact_id=contact_id,
        )

        return survey_response

