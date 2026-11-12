from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="AutomationSegmentOption")



@_attrs_define
class AutomationSegmentOption:
    """ 
        Attributes:
            check_minutes (int):
            id (str):
            name (str):
            eval_seconds (int | Unset):
     """

    check_minutes: int
    id: str
    name: str
    eval_seconds: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        check_minutes = self.check_minutes

        id = self.id

        name = self.name

        eval_seconds = self.eval_seconds


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "checkMinutes": check_minutes,
            "id": id,
            "name": name,
        })
        if eval_seconds is not UNSET:
            field_dict["evalSeconds"] = eval_seconds

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        check_minutes = d.pop("checkMinutes")

        id = d.pop("id")

        name = d.pop("name")

        eval_seconds = d.pop("evalSeconds", UNSET)

        automation_segment_option = cls(
            check_minutes=check_minutes,
            id=id,
            name=name,
            eval_seconds=eval_seconds,
        )

        return automation_segment_option

