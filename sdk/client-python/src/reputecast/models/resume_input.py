from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.resume_input_overdue import ResumeInputOverdue
from ..types import UNSET, Unset






T = TypeVar("T", bound="ResumeInput")



@_attrs_define
class ResumeInput:
    """ 
        Attributes:
            overdue (ResumeInputOverdue | Unset): Defaults to send
            spread_hours (int | Unset):
     """

    overdue: ResumeInputOverdue | Unset = UNSET
    spread_hours: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        overdue: str | Unset = UNSET
        if not isinstance(self.overdue, Unset):
            overdue = self.overdue.value


        spread_hours = self.spread_hours


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if overdue is not UNSET:
            field_dict["overdue"] = overdue
        if spread_hours is not UNSET:
            field_dict["spreadHours"] = spread_hours

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        _overdue = d.pop("overdue", UNSET)
        overdue: ResumeInputOverdue | Unset
        if isinstance(_overdue,  Unset):
            overdue = UNSET
        else:
            overdue = ResumeInputOverdue(_overdue)




        spread_hours = d.pop("spreadHours", UNSET)

        resume_input = cls(
            overdue=overdue,
            spread_hours=spread_hours,
        )

        return resume_input

