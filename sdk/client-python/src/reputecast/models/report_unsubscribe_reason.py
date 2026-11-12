from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.report_unsubscribe_reason_key import ReportUnsubscribeReasonKey
from ..models.report_unsubscribe_reason_tone import ReportUnsubscribeReasonTone






T = TypeVar("T", bound="ReportUnsubscribeReason")



@_attrs_define
class ReportUnsubscribeReason:
    """ 
        Attributes:
            count (int):
            key (ReportUnsubscribeReasonKey):
            label (str):
            share (float): Share of unsubscribes, percent
            tone (ReportUnsubscribeReasonTone):
     """

    count: int
    key: ReportUnsubscribeReasonKey
    label: str
    share: float
    tone: ReportUnsubscribeReasonTone





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        key = self.key.value

        label = self.label

        share = self.share

        tone = self.tone.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "key": key,
            "label": label,
            "share": share,
            "tone": tone,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        key = ReportUnsubscribeReasonKey(d.pop("key"))




        label = d.pop("label")

        share = d.pop("share")

        tone = ReportUnsubscribeReasonTone(d.pop("tone"))




        report_unsubscribe_reason = cls(
            count=count,
            key=key,
            label=label,
            share=share,
            tone=tone,
        )

        return report_unsubscribe_reason

