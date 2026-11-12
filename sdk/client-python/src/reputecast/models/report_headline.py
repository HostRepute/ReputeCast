from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.report_provider_count import ReportProviderCount





T = TypeVar("T", bound="ReportHeadline")



@_attrs_define
class ReportHeadline:
    """ 
        Attributes:
            bot_clicks (int):
            bounces (int):
            clicks (int):
            complaints (int):
            delivered (int):
            hard_bounces (int):
            human_opens (int):
            machine_opens (int): Scanner opens, left out of every rate
            measurable (int):
            not_reported_by (list[str]):
            privacy_opens (int):
            rejected (int):
            sent (int):
            sent_by (list[ReportProviderCount]):
            soft_bounces (int):
            undelivered (int):
            unsubscribes (int):
            bounce_rate (float | Unset):
            click_rate (float | Unset):
            click_to_open (float | Unset):
            complaint_rate (float | Unset):
            complaint_rate_reported (float | Unset):
            delivered_rate (float | Unset):
            human_open_rate (float | Unset):
            human_open_rate_measurable (float | Unset):
            machine_open_rate (float | Unset):
            privacy_open_rate (float | Unset):
            unsubscribe_rate (float | Unset):
     """

    bot_clicks: int
    bounces: int
    clicks: int
    complaints: int
    delivered: int
    hard_bounces: int
    human_opens: int
    machine_opens: int
    measurable: int
    not_reported_by: list[str]
    privacy_opens: int
    rejected: int
    sent: int
    sent_by: list[ReportProviderCount]
    soft_bounces: int
    undelivered: int
    unsubscribes: int
    bounce_rate: float | Unset = UNSET
    click_rate: float | Unset = UNSET
    click_to_open: float | Unset = UNSET
    complaint_rate: float | Unset = UNSET
    complaint_rate_reported: float | Unset = UNSET
    delivered_rate: float | Unset = UNSET
    human_open_rate: float | Unset = UNSET
    human_open_rate_measurable: float | Unset = UNSET
    machine_open_rate: float | Unset = UNSET
    privacy_open_rate: float | Unset = UNSET
    unsubscribe_rate: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.report_provider_count import ReportProviderCount # noqa: PLC0415
        bot_clicks = self.bot_clicks

        bounces = self.bounces

        clicks = self.clicks

        complaints = self.complaints

        delivered = self.delivered

        hard_bounces = self.hard_bounces

        human_opens = self.human_opens

        machine_opens = self.machine_opens

        measurable = self.measurable

        not_reported_by = self.not_reported_by



        privacy_opens = self.privacy_opens

        rejected = self.rejected

        sent = self.sent

        sent_by = []
        for sent_by_item_data in self.sent_by:
            sent_by_item = sent_by_item_data.to_dict()
            sent_by.append(sent_by_item)



        soft_bounces = self.soft_bounces

        undelivered = self.undelivered

        unsubscribes = self.unsubscribes

        bounce_rate = self.bounce_rate

        click_rate = self.click_rate

        click_to_open = self.click_to_open

        complaint_rate = self.complaint_rate

        complaint_rate_reported = self.complaint_rate_reported

        delivered_rate = self.delivered_rate

        human_open_rate = self.human_open_rate

        human_open_rate_measurable = self.human_open_rate_measurable

        machine_open_rate = self.machine_open_rate

        privacy_open_rate = self.privacy_open_rate

        unsubscribe_rate = self.unsubscribe_rate


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "botClicks": bot_clicks,
            "bounces": bounces,
            "clicks": clicks,
            "complaints": complaints,
            "delivered": delivered,
            "hardBounces": hard_bounces,
            "humanOpens": human_opens,
            "machineOpens": machine_opens,
            "measurable": measurable,
            "notReportedBy": not_reported_by,
            "privacyOpens": privacy_opens,
            "rejected": rejected,
            "sent": sent,
            "sentBy": sent_by,
            "softBounces": soft_bounces,
            "undelivered": undelivered,
            "unsubscribes": unsubscribes,
        })
        if bounce_rate is not UNSET:
            field_dict["bounceRate"] = bounce_rate
        if click_rate is not UNSET:
            field_dict["clickRate"] = click_rate
        if click_to_open is not UNSET:
            field_dict["clickToOpen"] = click_to_open
        if complaint_rate is not UNSET:
            field_dict["complaintRate"] = complaint_rate
        if complaint_rate_reported is not UNSET:
            field_dict["complaintRateReported"] = complaint_rate_reported
        if delivered_rate is not UNSET:
            field_dict["deliveredRate"] = delivered_rate
        if human_open_rate is not UNSET:
            field_dict["humanOpenRate"] = human_open_rate
        if human_open_rate_measurable is not UNSET:
            field_dict["humanOpenRateMeasurable"] = human_open_rate_measurable
        if machine_open_rate is not UNSET:
            field_dict["machineOpenRate"] = machine_open_rate
        if privacy_open_rate is not UNSET:
            field_dict["privacyOpenRate"] = privacy_open_rate
        if unsubscribe_rate is not UNSET:
            field_dict["unsubscribeRate"] = unsubscribe_rate

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.report_provider_count import ReportProviderCount # noqa: PLC0415
        d = dict(src_dict)
        bot_clicks = d.pop("botClicks")

        bounces = d.pop("bounces")

        clicks = d.pop("clicks")

        complaints = d.pop("complaints")

        delivered = d.pop("delivered")

        hard_bounces = d.pop("hardBounces")

        human_opens = d.pop("humanOpens")

        machine_opens = d.pop("machineOpens")

        measurable = d.pop("measurable")

        not_reported_by = cast(list[str], d.pop("notReportedBy"))


        privacy_opens = d.pop("privacyOpens")

        rejected = d.pop("rejected")

        sent = d.pop("sent")

        sent_by = []
        _sent_by = d.pop("sentBy")
        for sent_by_item_data in (_sent_by):
            sent_by_item = ReportProviderCount.from_dict(sent_by_item_data)



            sent_by.append(sent_by_item)


        soft_bounces = d.pop("softBounces")

        undelivered = d.pop("undelivered")

        unsubscribes = d.pop("unsubscribes")

        bounce_rate = d.pop("bounceRate", UNSET)

        click_rate = d.pop("clickRate", UNSET)

        click_to_open = d.pop("clickToOpen", UNSET)

        complaint_rate = d.pop("complaintRate", UNSET)

        complaint_rate_reported = d.pop("complaintRateReported", UNSET)

        delivered_rate = d.pop("deliveredRate", UNSET)

        human_open_rate = d.pop("humanOpenRate", UNSET)

        human_open_rate_measurable = d.pop("humanOpenRateMeasurable", UNSET)

        machine_open_rate = d.pop("machineOpenRate", UNSET)

        privacy_open_rate = d.pop("privacyOpenRate", UNSET)

        unsubscribe_rate = d.pop("unsubscribeRate", UNSET)

        report_headline = cls(
            bot_clicks=bot_clicks,
            bounces=bounces,
            clicks=clicks,
            complaints=complaints,
            delivered=delivered,
            hard_bounces=hard_bounces,
            human_opens=human_opens,
            machine_opens=machine_opens,
            measurable=measurable,
            not_reported_by=not_reported_by,
            privacy_opens=privacy_opens,
            rejected=rejected,
            sent=sent,
            sent_by=sent_by,
            soft_bounces=soft_bounces,
            undelivered=undelivered,
            unsubscribes=unsubscribes,
            bounce_rate=bounce_rate,
            click_rate=click_rate,
            click_to_open=click_to_open,
            complaint_rate=complaint_rate,
            complaint_rate_reported=complaint_rate_reported,
            delivered_rate=delivered_rate,
            human_open_rate=human_open_rate,
            human_open_rate_measurable=human_open_rate_measurable,
            machine_open_rate=machine_open_rate,
            privacy_open_rate=privacy_open_rate,
            unsubscribe_rate=unsubscribe_rate,
        )

        return report_headline

