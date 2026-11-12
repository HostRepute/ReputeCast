from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.report_bounce_row_suppression import ReportBounceRowSuppression
from ..models.report_bounce_row_tone import ReportBounceRowTone
from ..models.report_bounce_row_type import ReportBounceRowType
from typing import cast

if TYPE_CHECKING:
  from ..models.report_code_count import ReportCodeCount





T = TypeVar("T", bound="ReportBounceRow")



@_attrs_define
class ReportBounceRow:
    """ 
        Attributes:
            codes (list[ReportCodeCount]):
            count (int):
            label (str):
            suppression (ReportBounceRowSuppression):
            tone (ReportBounceRowTone):
            type_ (ReportBounceRowType):
            unsuppressed (int):
     """

    codes: list[ReportCodeCount]
    count: int
    label: str
    suppression: ReportBounceRowSuppression
    tone: ReportBounceRowTone
    type_: ReportBounceRowType
    unsuppressed: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.report_code_count import ReportCodeCount # noqa: PLC0415
        codes = []
        for codes_item_data in self.codes:
            codes_item = codes_item_data.to_dict()
            codes.append(codes_item)



        count = self.count

        label = self.label

        suppression = self.suppression.value

        tone = self.tone.value

        type_ = self.type_.value

        unsuppressed = self.unsuppressed


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "codes": codes,
            "count": count,
            "label": label,
            "suppression": suppression,
            "tone": tone,
            "type": type_,
            "unsuppressed": unsuppressed,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.report_code_count import ReportCodeCount # noqa: PLC0415
        d = dict(src_dict)
        codes = []
        _codes = d.pop("codes")
        for codes_item_data in (_codes):
            codes_item = ReportCodeCount.from_dict(codes_item_data)



            codes.append(codes_item)


        count = d.pop("count")

        label = d.pop("label")

        suppression = ReportBounceRowSuppression(d.pop("suppression"))




        tone = ReportBounceRowTone(d.pop("tone"))




        type_ = ReportBounceRowType(d.pop("type"))




        unsuppressed = d.pop("unsuppressed")

        report_bounce_row = cls(
            codes=codes,
            count=count,
            label=label,
            suppression=suppression,
            tone=tone,
            type_=type_,
            unsuppressed=unsuppressed,
        )

        return report_bounce_row

