from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.report_provider_column_color import ReportProviderColumnColor
from ..models.report_provider_column_complaint_rate_na import ReportProviderColumnComplaintRateNa
from ..models.report_provider_column_events import ReportProviderColumnEvents
from ..models.report_provider_column_tracked_by import ReportProviderColumnTrackedBy
from ..types import UNSET, Unset






T = TypeVar("T", bound="ReportProviderColumn")



@_attrs_define
class ReportProviderColumn:
    """ 
        Attributes:
            adapter (str):
            color (ReportProviderColumnColor):
            delivered (int):
            delivery_inferred (bool):
            engaged_first (bool):
            events (ReportProviderColumnEvents):
            measurable (int):
            name (str):
            percent (float):
            provider_id (str):
            sent (int):
            short (str):
            tracked_by (ReportProviderColumnTrackedBy):
            bounce_rate (float | Unset):
            click_rate (float | Unset):
            complaint_rate (float | Unset):
            complaint_rate_na (ReportProviderColumnComplaintRateNa | Unset):
            cost (float | Unset): Dollars
            delivered_rate (float | Unset):
            human_open_rate (float | Unset):
            human_open_rate_measurable (float | Unset):
            machine_open_rate (float | Unset):
            median_delivery_ms (int | Unset):
            privacy_open_rate (float | Unset):
            unsubscribe_rate (float | Unset):
            warmup_day (int | Unset):
     """

    adapter: str
    color: ReportProviderColumnColor
    delivered: int
    delivery_inferred: bool
    engaged_first: bool
    events: ReportProviderColumnEvents
    measurable: int
    name: str
    percent: float
    provider_id: str
    sent: int
    short: str
    tracked_by: ReportProviderColumnTrackedBy
    bounce_rate: float | Unset = UNSET
    click_rate: float | Unset = UNSET
    complaint_rate: float | Unset = UNSET
    complaint_rate_na: ReportProviderColumnComplaintRateNa | Unset = UNSET
    cost: float | Unset = UNSET
    delivered_rate: float | Unset = UNSET
    human_open_rate: float | Unset = UNSET
    human_open_rate_measurable: float | Unset = UNSET
    machine_open_rate: float | Unset = UNSET
    median_delivery_ms: int | Unset = UNSET
    privacy_open_rate: float | Unset = UNSET
    unsubscribe_rate: float | Unset = UNSET
    warmup_day: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        adapter = self.adapter

        color = self.color.value

        delivered = self.delivered

        delivery_inferred = self.delivery_inferred

        engaged_first = self.engaged_first

        events = self.events.value

        measurable = self.measurable

        name = self.name

        percent = self.percent

        provider_id = self.provider_id

        sent = self.sent

        short = self.short

        tracked_by = self.tracked_by.value

        bounce_rate = self.bounce_rate

        click_rate = self.click_rate

        complaint_rate = self.complaint_rate

        complaint_rate_na: str | Unset = UNSET
        if not isinstance(self.complaint_rate_na, Unset):
            complaint_rate_na = self.complaint_rate_na.value


        cost = self.cost

        delivered_rate = self.delivered_rate

        human_open_rate = self.human_open_rate

        human_open_rate_measurable = self.human_open_rate_measurable

        machine_open_rate = self.machine_open_rate

        median_delivery_ms = self.median_delivery_ms

        privacy_open_rate = self.privacy_open_rate

        unsubscribe_rate = self.unsubscribe_rate

        warmup_day = self.warmup_day


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "color": color,
            "delivered": delivered,
            "deliveryInferred": delivery_inferred,
            "engagedFirst": engaged_first,
            "events": events,
            "measurable": measurable,
            "name": name,
            "percent": percent,
            "providerId": provider_id,
            "sent": sent,
            "short": short,
            "trackedBy": tracked_by,
        })
        if bounce_rate is not UNSET:
            field_dict["bounceRate"] = bounce_rate
        if click_rate is not UNSET:
            field_dict["clickRate"] = click_rate
        if complaint_rate is not UNSET:
            field_dict["complaintRate"] = complaint_rate
        if complaint_rate_na is not UNSET:
            field_dict["complaintRateNa"] = complaint_rate_na
        if cost is not UNSET:
            field_dict["cost"] = cost
        if delivered_rate is not UNSET:
            field_dict["deliveredRate"] = delivered_rate
        if human_open_rate is not UNSET:
            field_dict["humanOpenRate"] = human_open_rate
        if human_open_rate_measurable is not UNSET:
            field_dict["humanOpenRateMeasurable"] = human_open_rate_measurable
        if machine_open_rate is not UNSET:
            field_dict["machineOpenRate"] = machine_open_rate
        if median_delivery_ms is not UNSET:
            field_dict["medianDeliveryMs"] = median_delivery_ms
        if privacy_open_rate is not UNSET:
            field_dict["privacyOpenRate"] = privacy_open_rate
        if unsubscribe_rate is not UNSET:
            field_dict["unsubscribeRate"] = unsubscribe_rate
        if warmup_day is not UNSET:
            field_dict["warmupDay"] = warmup_day

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        adapter = d.pop("adapter")

        color = ReportProviderColumnColor(d.pop("color"))




        delivered = d.pop("delivered")

        delivery_inferred = d.pop("deliveryInferred")

        engaged_first = d.pop("engagedFirst")

        events = ReportProviderColumnEvents(d.pop("events"))




        measurable = d.pop("measurable")

        name = d.pop("name")

        percent = d.pop("percent")

        provider_id = d.pop("providerId")

        sent = d.pop("sent")

        short = d.pop("short")

        tracked_by = ReportProviderColumnTrackedBy(d.pop("trackedBy"))




        bounce_rate = d.pop("bounceRate", UNSET)

        click_rate = d.pop("clickRate", UNSET)

        complaint_rate = d.pop("complaintRate", UNSET)

        _complaint_rate_na = d.pop("complaintRateNa", UNSET)
        complaint_rate_na: ReportProviderColumnComplaintRateNa | Unset
        if isinstance(_complaint_rate_na,  Unset):
            complaint_rate_na = UNSET
        else:
            complaint_rate_na = ReportProviderColumnComplaintRateNa(_complaint_rate_na)




        cost = d.pop("cost", UNSET)

        delivered_rate = d.pop("deliveredRate", UNSET)

        human_open_rate = d.pop("humanOpenRate", UNSET)

        human_open_rate_measurable = d.pop("humanOpenRateMeasurable", UNSET)

        machine_open_rate = d.pop("machineOpenRate", UNSET)

        median_delivery_ms = d.pop("medianDeliveryMs", UNSET)

        privacy_open_rate = d.pop("privacyOpenRate", UNSET)

        unsubscribe_rate = d.pop("unsubscribeRate", UNSET)

        warmup_day = d.pop("warmupDay", UNSET)

        report_provider_column = cls(
            adapter=adapter,
            color=color,
            delivered=delivered,
            delivery_inferred=delivery_inferred,
            engaged_first=engaged_first,
            events=events,
            measurable=measurable,
            name=name,
            percent=percent,
            provider_id=provider_id,
            sent=sent,
            short=short,
            tracked_by=tracked_by,
            bounce_rate=bounce_rate,
            click_rate=click_rate,
            complaint_rate=complaint_rate,
            complaint_rate_na=complaint_rate_na,
            cost=cost,
            delivered_rate=delivered_rate,
            human_open_rate=human_open_rate,
            human_open_rate_measurable=human_open_rate_measurable,
            machine_open_rate=machine_open_rate,
            median_delivery_ms=median_delivery_ms,
            privacy_open_rate=privacy_open_rate,
            unsubscribe_rate=unsubscribe_rate,
            warmup_day=warmup_day,
        )

        return report_provider_column

