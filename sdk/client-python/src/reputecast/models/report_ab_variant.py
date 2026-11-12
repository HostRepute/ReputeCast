from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="ReportABVariant")



@_attrs_define
class ReportABVariant:
    """ 
        Attributes:
            conversions (int):
            default (bool):
            eligible (int):
            key (str):
            label (str): The subject line, or what differs
            recipients (int):
            remaining (int):
            sample_percent (float):
            winner (bool):
            click_rate (float | Unset):
            complaint_rate (float | Unset):
            decision_rate (float | Unset):
            human_open_rate (float | Unset):
            metric_rate (float | Unset):
     """

    conversions: int
    default: bool
    eligible: int
    key: str
    label: str
    recipients: int
    remaining: int
    sample_percent: float
    winner: bool
    click_rate: float | Unset = UNSET
    complaint_rate: float | Unset = UNSET
    decision_rate: float | Unset = UNSET
    human_open_rate: float | Unset = UNSET
    metric_rate: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        conversions = self.conversions

        default = self.default

        eligible = self.eligible

        key = self.key

        label = self.label

        recipients = self.recipients

        remaining = self.remaining

        sample_percent = self.sample_percent

        winner = self.winner

        click_rate = self.click_rate

        complaint_rate = self.complaint_rate

        decision_rate = self.decision_rate

        human_open_rate = self.human_open_rate

        metric_rate = self.metric_rate


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "conversions": conversions,
            "default": default,
            "eligible": eligible,
            "key": key,
            "label": label,
            "recipients": recipients,
            "remaining": remaining,
            "samplePercent": sample_percent,
            "winner": winner,
        })
        if click_rate is not UNSET:
            field_dict["clickRate"] = click_rate
        if complaint_rate is not UNSET:
            field_dict["complaintRate"] = complaint_rate
        if decision_rate is not UNSET:
            field_dict["decisionRate"] = decision_rate
        if human_open_rate is not UNSET:
            field_dict["humanOpenRate"] = human_open_rate
        if metric_rate is not UNSET:
            field_dict["metricRate"] = metric_rate

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        conversions = d.pop("conversions")

        default = d.pop("default")

        eligible = d.pop("eligible")

        key = d.pop("key")

        label = d.pop("label")

        recipients = d.pop("recipients")

        remaining = d.pop("remaining")

        sample_percent = d.pop("samplePercent")

        winner = d.pop("winner")

        click_rate = d.pop("clickRate", UNSET)

        complaint_rate = d.pop("complaintRate", UNSET)

        decision_rate = d.pop("decisionRate", UNSET)

        human_open_rate = d.pop("humanOpenRate", UNSET)

        metric_rate = d.pop("metricRate", UNSET)

        report_ab_variant = cls(
            conversions=conversions,
            default=default,
            eligible=eligible,
            key=key,
            label=label,
            recipients=recipients,
            remaining=remaining,
            sample_percent=sample_percent,
            winner=winner,
            click_rate=click_rate,
            complaint_rate=complaint_rate,
            decision_rate=decision_rate,
            human_open_rate=human_open_rate,
            metric_rate=metric_rate,
        )

        return report_ab_variant

