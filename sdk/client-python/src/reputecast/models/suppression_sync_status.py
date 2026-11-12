from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.suppression_reason_count import SuppressionReasonCount
  from ..models.suppression_sync_row import SuppressionSyncRow





T = TypeVar("T", bound="SuppressionSyncStatus")



@_attrs_define
class SuppressionSyncStatus:
    """ 
        Attributes:
            kept (list[SuppressionReasonCount]):
            kept_all_mail (int):
            marketing_only (int):
            pushed (list[SuppressionReasonCount]):
            pushed_total (int):
            reasons (list[SuppressionReasonCount]):
            rows (list[SuppressionSyncRow]):
            total (int):
     """

    kept: list[SuppressionReasonCount]
    kept_all_mail: int
    marketing_only: int
    pushed: list[SuppressionReasonCount]
    pushed_total: int
    reasons: list[SuppressionReasonCount]
    rows: list[SuppressionSyncRow]
    total: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.suppression_reason_count import SuppressionReasonCount # noqa: PLC0415
        from ..models.suppression_sync_row import SuppressionSyncRow # noqa: PLC0415
        kept = []
        for kept_item_data in self.kept:
            kept_item = kept_item_data.to_dict()
            kept.append(kept_item)



        kept_all_mail = self.kept_all_mail

        marketing_only = self.marketing_only

        pushed = []
        for pushed_item_data in self.pushed:
            pushed_item = pushed_item_data.to_dict()
            pushed.append(pushed_item)



        pushed_total = self.pushed_total

        reasons = []
        for reasons_item_data in self.reasons:
            reasons_item = reasons_item_data.to_dict()
            reasons.append(reasons_item)



        rows = []
        for rows_item_data in self.rows:
            rows_item = rows_item_data.to_dict()
            rows.append(rows_item)



        total = self.total


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kept": kept,
            "keptAllMail": kept_all_mail,
            "marketingOnly": marketing_only,
            "pushed": pushed,
            "pushedTotal": pushed_total,
            "reasons": reasons,
            "rows": rows,
            "total": total,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.suppression_reason_count import SuppressionReasonCount # noqa: PLC0415
        from ..models.suppression_sync_row import SuppressionSyncRow # noqa: PLC0415
        d = dict(src_dict)
        kept = []
        _kept = d.pop("kept")
        for kept_item_data in (_kept):
            kept_item = SuppressionReasonCount.from_dict(kept_item_data)



            kept.append(kept_item)


        kept_all_mail = d.pop("keptAllMail")

        marketing_only = d.pop("marketingOnly")

        pushed = []
        _pushed = d.pop("pushed")
        for pushed_item_data in (_pushed):
            pushed_item = SuppressionReasonCount.from_dict(pushed_item_data)



            pushed.append(pushed_item)


        pushed_total = d.pop("pushedTotal")

        reasons = []
        _reasons = d.pop("reasons")
        for reasons_item_data in (_reasons):
            reasons_item = SuppressionReasonCount.from_dict(reasons_item_data)



            reasons.append(reasons_item)


        rows = []
        _rows = d.pop("rows")
        for rows_item_data in (_rows):
            rows_item = SuppressionSyncRow.from_dict(rows_item_data)



            rows.append(rows_item)


        total = d.pop("total")

        suppression_sync_status = cls(
            kept=kept,
            kept_all_mail=kept_all_mail,
            marketing_only=marketing_only,
            pushed=pushed,
            pushed_total=pushed_total,
            reasons=reasons,
            rows=rows,
            total=total,
        )

        return suppression_sync_status

