from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.sponsor_creative import SponsorCreative
  from ..models.sponsor_report_link import SponsorReportLink





T = TypeVar("T", bound="SponsorReport")



@_attrs_define
class SponsorReport:
    """ 
        Attributes:
            booking_id (str):
            click_rate (float):
            click_to_open (float):
            clicks (int):
            creative (SponsorCreative):
            delivered (int):
            final (bool):
            human_open_rate (float):
            human_opens (int):
            initials (str):
            issue (int):
            issue_on (datetime.datetime):
            label (str):
            link_campaign (str):
            link_host (str):
            links (list[SponsorReportLink]):
            list_name (str):
            placement (str):
            slot_name (str):
            sponsor (str):
            sponsor_name (str):
            unique_clickers (int):
            final_at (datetime.datetime | Unset):
            sent_at (datetime.datetime | Unset):
            visibility (str | Unset):
     """

    booking_id: str
    click_rate: float
    click_to_open: float
    clicks: int
    creative: SponsorCreative
    delivered: int
    final: bool
    human_open_rate: float
    human_opens: int
    initials: str
    issue: int
    issue_on: datetime.datetime
    label: str
    link_campaign: str
    link_host: str
    links: list[SponsorReportLink]
    list_name: str
    placement: str
    slot_name: str
    sponsor: str
    sponsor_name: str
    unique_clickers: int
    final_at: datetime.datetime | Unset = UNSET
    sent_at: datetime.datetime | Unset = UNSET
    visibility: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sponsor_creative import SponsorCreative # noqa: PLC0415
        from ..models.sponsor_report_link import SponsorReportLink # noqa: PLC0415
        booking_id = self.booking_id

        click_rate = self.click_rate

        click_to_open = self.click_to_open

        clicks = self.clicks

        creative = self.creative.to_dict()

        delivered = self.delivered

        final = self.final

        human_open_rate = self.human_open_rate

        human_opens = self.human_opens

        initials = self.initials

        issue = self.issue

        issue_on = self.issue_on.isoformat()

        label = self.label

        link_campaign = self.link_campaign

        link_host = self.link_host

        links = []
        for links_item_data in self.links:
            links_item = links_item_data.to_dict()
            links.append(links_item)



        list_name = self.list_name

        placement = self.placement

        slot_name = self.slot_name

        sponsor = self.sponsor

        sponsor_name = self.sponsor_name

        unique_clickers = self.unique_clickers

        final_at: str | Unset = UNSET
        if not isinstance(self.final_at, Unset):
            final_at = self.final_at.isoformat()

        sent_at: str | Unset = UNSET
        if not isinstance(self.sent_at, Unset):
            sent_at = self.sent_at.isoformat()

        visibility = self.visibility


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bookingId": booking_id,
            "clickRate": click_rate,
            "clickToOpen": click_to_open,
            "clicks": clicks,
            "creative": creative,
            "delivered": delivered,
            "final": final,
            "humanOpenRate": human_open_rate,
            "humanOpens": human_opens,
            "initials": initials,
            "issue": issue,
            "issueOn": issue_on,
            "label": label,
            "linkCampaign": link_campaign,
            "linkHost": link_host,
            "links": links,
            "listName": list_name,
            "placement": placement,
            "slotName": slot_name,
            "sponsor": sponsor,
            "sponsorName": sponsor_name,
            "uniqueClickers": unique_clickers,
        })
        if final_at is not UNSET:
            field_dict["finalAt"] = final_at
        if sent_at is not UNSET:
            field_dict["sentAt"] = sent_at
        if visibility is not UNSET:
            field_dict["visibility"] = visibility

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sponsor_creative import SponsorCreative # noqa: PLC0415
        from ..models.sponsor_report_link import SponsorReportLink # noqa: PLC0415
        d = dict(src_dict)
        booking_id = d.pop("bookingId")

        click_rate = d.pop("clickRate")

        click_to_open = d.pop("clickToOpen")

        clicks = d.pop("clicks")

        creative = SponsorCreative.from_dict(d.pop("creative"))




        delivered = d.pop("delivered")

        final = d.pop("final")

        human_open_rate = d.pop("humanOpenRate")

        human_opens = d.pop("humanOpens")

        initials = d.pop("initials")

        issue = d.pop("issue")

        issue_on = datetime.datetime.fromisoformat(d.pop("issueOn"))




        label = d.pop("label")

        link_campaign = d.pop("linkCampaign")

        link_host = d.pop("linkHost")

        links = []
        _links = d.pop("links")
        for links_item_data in (_links):
            links_item = SponsorReportLink.from_dict(links_item_data)



            links.append(links_item)


        list_name = d.pop("listName")

        placement = d.pop("placement")

        slot_name = d.pop("slotName")

        sponsor = d.pop("sponsor")

        sponsor_name = d.pop("sponsorName")

        unique_clickers = d.pop("uniqueClickers")

        _final_at = d.pop("finalAt", UNSET)
        final_at: datetime.datetime | Unset
        if isinstance(_final_at,  Unset):
            final_at = UNSET
        else:
            final_at = datetime.datetime.fromisoformat(_final_at)




        _sent_at = d.pop("sentAt", UNSET)
        sent_at: datetime.datetime | Unset
        if isinstance(_sent_at,  Unset):
            sent_at = UNSET
        else:
            sent_at = datetime.datetime.fromisoformat(_sent_at)




        visibility = d.pop("visibility", UNSET)

        sponsor_report = cls(
            booking_id=booking_id,
            click_rate=click_rate,
            click_to_open=click_to_open,
            clicks=clicks,
            creative=creative,
            delivered=delivered,
            final=final,
            human_open_rate=human_open_rate,
            human_opens=human_opens,
            initials=initials,
            issue=issue,
            issue_on=issue_on,
            label=label,
            link_campaign=link_campaign,
            link_host=link_host,
            links=links,
            list_name=list_name,
            placement=placement,
            slot_name=slot_name,
            sponsor=sponsor,
            sponsor_name=sponsor_name,
            unique_clickers=unique_clickers,
            final_at=final_at,
            sent_at=sent_at,
            visibility=visibility,
        )

        return sponsor_report

