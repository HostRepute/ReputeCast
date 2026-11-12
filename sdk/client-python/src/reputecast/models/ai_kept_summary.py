from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.ai_kept_summary_view import AIKeptSummaryView





T = TypeVar("T", bound="AIKeptSummary")



@_attrs_define
class AIKeptSummary:
    """ 
        Attributes:
            host_repute (bool):
            kept (AIKeptSummaryView | Unset):
     """

    host_repute: bool
    kept: AIKeptSummaryView | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.ai_kept_summary_view import AIKeptSummaryView # noqa: PLC0415
        host_repute = self.host_repute

        kept: dict[str, Any] | Unset = UNSET
        if not isinstance(self.kept, Unset):
            kept = self.kept.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "hostRepute": host_repute,
        })
        if kept is not UNSET:
            field_dict["kept"] = kept

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.ai_kept_summary_view import AIKeptSummaryView # noqa: PLC0415
        d = dict(src_dict)
        host_repute = d.pop("hostRepute")

        _kept = d.pop("kept", UNSET)
        kept: AIKeptSummaryView | Unset
        if isinstance(_kept,  Unset):
            kept = UNSET
        else:
            kept = AIKeptSummaryView.from_dict(_kept)




        ai_kept_summary = cls(
            host_repute=host_repute,
            kept=kept,
        )

        return ai_kept_summary

