from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.report_unsubscribe_reason import ReportUnsubscribeReason





T = TypeVar("T", bound="ReportUnsubscribes")



@_attrs_define
class ReportUnsubscribes:
    """ 
        Attributes:
            chose_fewer (int):
            reasons (list[ReportUnsubscribeReason]):
            total (int):
     """

    chose_fewer: int
    reasons: list[ReportUnsubscribeReason]
    total: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.report_unsubscribe_reason import ReportUnsubscribeReason # noqa: PLC0415
        chose_fewer = self.chose_fewer

        reasons = []
        for reasons_item_data in self.reasons:
            reasons_item = reasons_item_data.to_dict()
            reasons.append(reasons_item)



        total = self.total


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "choseFewer": chose_fewer,
            "reasons": reasons,
            "total": total,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.report_unsubscribe_reason import ReportUnsubscribeReason # noqa: PLC0415
        d = dict(src_dict)
        chose_fewer = d.pop("choseFewer")

        reasons = []
        _reasons = d.pop("reasons")
        for reasons_item_data in (_reasons):
            reasons_item = ReportUnsubscribeReason.from_dict(reasons_item_data)



            reasons.append(reasons_item)


        total = d.pop("total")

        report_unsubscribes = cls(
            chose_fewer=chose_fewer,
            reasons=reasons,
            total=total,
        )

        return report_unsubscribes

