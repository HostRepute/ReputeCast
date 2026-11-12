from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SignupFormSchedule")



@_attrs_define
class SignupFormSchedule:
    """ 
        Attributes:
            ends_at (datetime.datetime | Unset):
            starts_at (datetime.datetime | Unset):
     """

    ends_at: datetime.datetime | Unset = UNSET
    starts_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        ends_at: str | Unset = UNSET
        if not isinstance(self.ends_at, Unset):
            ends_at = self.ends_at.isoformat()

        starts_at: str | Unset = UNSET
        if not isinstance(self.starts_at, Unset):
            starts_at = self.starts_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if ends_at is not UNSET:
            field_dict["endsAt"] = ends_at
        if starts_at is not UNSET:
            field_dict["startsAt"] = starts_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        _ends_at = d.pop("endsAt", UNSET)
        ends_at: datetime.datetime | Unset
        if isinstance(_ends_at,  Unset):
            ends_at = UNSET
        else:
            ends_at = datetime.datetime.fromisoformat(_ends_at)




        _starts_at = d.pop("startsAt", UNSET)
        starts_at: datetime.datetime | Unset
        if isinstance(_starts_at,  Unset):
            starts_at = UNSET
        else:
            starts_at = datetime.datetime.fromisoformat(_starts_at)




        signup_form_schedule = cls(
            ends_at=ends_at,
            starts_at=starts_at,
        )

        return signup_form_schedule

