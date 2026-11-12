from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ReportFinalAlert")



@_attrs_define
class ReportFinalAlert:
    """ 
        Attributes:
            on (bool):
            final_at (datetime.datetime | Unset):
     """

    on: bool
    final_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        on = self.on

        final_at: str | Unset = UNSET
        if not isinstance(self.final_at, Unset):
            final_at = self.final_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "on": on,
        })
        if final_at is not UNSET:
            field_dict["finalAt"] = final_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        on = d.pop("on")

        _final_at = d.pop("finalAt", UNSET)
        final_at: datetime.datetime | Unset
        if isinstance(_final_at,  Unset):
            final_at = UNSET
        else:
            final_at = datetime.datetime.fromisoformat(_final_at)




        report_final_alert = cls(
            on=on,
            final_at=final_at,
        )

        return report_final_alert

