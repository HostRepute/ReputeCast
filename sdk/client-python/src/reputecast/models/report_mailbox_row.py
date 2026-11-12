from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.report_mailbox_row_complaint_rate_na import ReportMailboxRowComplaintRateNa
from ..models.report_mailbox_row_key import ReportMailboxRowKey
from ..types import UNSET, Unset






T = TypeVar("T", bound="ReportMailboxRow")



@_attrs_define
class ReportMailboxRow:
    """ 
        Attributes:
            delivered (int):
            key (ReportMailboxRowKey):
            measurable (int):
            name (str):
            sent (int):
            bounce_rate (float | Unset):
            click_rate (float | Unset):
            complaint_rate (float | Unset):
            complaint_rate_na (ReportMailboxRowComplaintRateNa | Unset):
            delivered_rate (float | Unset):
            human_open_rate (float | Unset):
            human_open_rate_measurable (float | Unset):
            machine_open_rate (float | Unset):
            privacy_open_rate (float | Unset):
            unsubscribe_rate (float | Unset):
     """

    delivered: int
    key: ReportMailboxRowKey
    measurable: int
    name: str
    sent: int
    bounce_rate: float | Unset = UNSET
    click_rate: float | Unset = UNSET
    complaint_rate: float | Unset = UNSET
    complaint_rate_na: ReportMailboxRowComplaintRateNa | Unset = UNSET
    delivered_rate: float | Unset = UNSET
    human_open_rate: float | Unset = UNSET
    human_open_rate_measurable: float | Unset = UNSET
    machine_open_rate: float | Unset = UNSET
    privacy_open_rate: float | Unset = UNSET
    unsubscribe_rate: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        delivered = self.delivered

        key = self.key.value

        measurable = self.measurable

        name = self.name

        sent = self.sent

        bounce_rate = self.bounce_rate

        click_rate = self.click_rate

        complaint_rate = self.complaint_rate

        complaint_rate_na: str | Unset = UNSET
        if not isinstance(self.complaint_rate_na, Unset):
            complaint_rate_na = self.complaint_rate_na.value


        delivered_rate = self.delivered_rate

        human_open_rate = self.human_open_rate

        human_open_rate_measurable = self.human_open_rate_measurable

        machine_open_rate = self.machine_open_rate

        privacy_open_rate = self.privacy_open_rate

        unsubscribe_rate = self.unsubscribe_rate


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "delivered": delivered,
            "key": key,
            "measurable": measurable,
            "name": name,
            "sent": sent,
        })
        if bounce_rate is not UNSET:
            field_dict["bounceRate"] = bounce_rate
        if click_rate is not UNSET:
            field_dict["clickRate"] = click_rate
        if complaint_rate is not UNSET:
            field_dict["complaintRate"] = complaint_rate
        if complaint_rate_na is not UNSET:
            field_dict["complaintRateNa"] = complaint_rate_na
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
        d = dict(src_dict)
        delivered = d.pop("delivered")

        key = ReportMailboxRowKey(d.pop("key"))




        measurable = d.pop("measurable")

        name = d.pop("name")

        sent = d.pop("sent")

        bounce_rate = d.pop("bounceRate", UNSET)

        click_rate = d.pop("clickRate", UNSET)

        complaint_rate = d.pop("complaintRate", UNSET)

        _complaint_rate_na = d.pop("complaintRateNa", UNSET)
        complaint_rate_na: ReportMailboxRowComplaintRateNa | Unset
        if isinstance(_complaint_rate_na,  Unset):
            complaint_rate_na = UNSET
        else:
            complaint_rate_na = ReportMailboxRowComplaintRateNa(_complaint_rate_na)




        delivered_rate = d.pop("deliveredRate", UNSET)

        human_open_rate = d.pop("humanOpenRate", UNSET)

        human_open_rate_measurable = d.pop("humanOpenRateMeasurable", UNSET)

        machine_open_rate = d.pop("machineOpenRate", UNSET)

        privacy_open_rate = d.pop("privacyOpenRate", UNSET)

        unsubscribe_rate = d.pop("unsubscribeRate", UNSET)

        report_mailbox_row = cls(
            delivered=delivered,
            key=key,
            measurable=measurable,
            name=name,
            sent=sent,
            bounce_rate=bounce_rate,
            click_rate=click_rate,
            complaint_rate=complaint_rate,
            complaint_rate_na=complaint_rate_na,
            delivered_rate=delivered_rate,
            human_open_rate=human_open_rate,
            human_open_rate_measurable=human_open_rate_measurable,
            machine_open_rate=machine_open_rate,
            privacy_open_rate=privacy_open_rate,
            unsubscribe_rate=unsubscribe_rate,
        )

        return report_mailbox_row

