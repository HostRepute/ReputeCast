from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sponsor_booking_detail_creative_status import SponsorBookingDetailCreativeStatus
from ..models.sponsor_booking_detail_status import SponsorBookingDetailStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.sponsor_booking_event import SponsorBookingEvent
  from ..models.sponsor_campaign_ref import SponsorCampaignRef
  from ..models.sponsor_creative import SponsorCreative
  from ..models.sponsor_item import SponsorItem
  from ..models.sponsor_payment import SponsorPayment
  from ..models.sponsor_propose import SponsorPropose
  from ..models.sponsor_ref import SponsorRef
  from ..models.sponsor_report import SponsorReport
  from ..models.sponsor_share_link import SponsorShareLink





T = TypeVar("T", bound="SponsorBookingDetail")



@_attrs_define
class SponsorBookingDetail:
    """ 
        Attributes:
            activity (list[SponsorBookingEvent]):
            can_manage (bool):
            can_share (bool):
            creative (SponsorCreative):
            creative_status (SponsorBookingDetailCreativeStatus):
            currency (str):
            helpers (list[str]):
            id (str):
            issue (int):
            issue_on (datetime.datetime):
            label (str):
            library (list[SponsorRef]):
            list_name (str):
            notes (str):
            payment (SponsorPayment):
            placement (str):
            price_cents (int):
            shares (list[SponsorShareLink]):
            slot_id (str):
            slot_name (str):
            slot_price_cents (int):
            status (SponsorBookingDetailStatus):
            campaign (SponsorCampaignRef | Unset):
            canceled_at (datetime.datetime | Unset):
            creative_due (datetime.datetime | Unset):
            hold_until (datetime.datetime | Unset):
            proposal (SponsorPropose | Unset):
            ready_at (datetime.datetime | Unset):
            report (SponsorReport | Unset):
            sponsor (SponsorItem | Unset):
     """

    activity: list[SponsorBookingEvent]
    can_manage: bool
    can_share: bool
    creative: SponsorCreative
    creative_status: SponsorBookingDetailCreativeStatus
    currency: str
    helpers: list[str]
    id: str
    issue: int
    issue_on: datetime.datetime
    label: str
    library: list[SponsorRef]
    list_name: str
    notes: str
    payment: SponsorPayment
    placement: str
    price_cents: int
    shares: list[SponsorShareLink]
    slot_id: str
    slot_name: str
    slot_price_cents: int
    status: SponsorBookingDetailStatus
    campaign: SponsorCampaignRef | Unset = UNSET
    canceled_at: datetime.datetime | Unset = UNSET
    creative_due: datetime.datetime | Unset = UNSET
    hold_until: datetime.datetime | Unset = UNSET
    proposal: SponsorPropose | Unset = UNSET
    ready_at: datetime.datetime | Unset = UNSET
    report: SponsorReport | Unset = UNSET
    sponsor: SponsorItem | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sponsor_booking_event import SponsorBookingEvent # noqa: PLC0415
        from ..models.sponsor_campaign_ref import SponsorCampaignRef # noqa: PLC0415
        from ..models.sponsor_creative import SponsorCreative # noqa: PLC0415
        from ..models.sponsor_item import SponsorItem # noqa: PLC0415
        from ..models.sponsor_payment import SponsorPayment # noqa: PLC0415
        from ..models.sponsor_propose import SponsorPropose # noqa: PLC0415
        from ..models.sponsor_ref import SponsorRef # noqa: PLC0415
        from ..models.sponsor_report import SponsorReport # noqa: PLC0415
        from ..models.sponsor_share_link import SponsorShareLink # noqa: PLC0415
        activity = []
        for activity_item_data in self.activity:
            activity_item = activity_item_data.to_dict()
            activity.append(activity_item)



        can_manage = self.can_manage

        can_share = self.can_share

        creative = self.creative.to_dict()

        creative_status = self.creative_status.value

        currency = self.currency

        helpers = self.helpers



        id = self.id

        issue = self.issue

        issue_on = self.issue_on.isoformat()

        label = self.label

        library = []
        for library_item_data in self.library:
            library_item = library_item_data.to_dict()
            library.append(library_item)



        list_name = self.list_name

        notes = self.notes

        payment = self.payment.to_dict()

        placement = self.placement

        price_cents = self.price_cents

        shares = []
        for shares_item_data in self.shares:
            shares_item = shares_item_data.to_dict()
            shares.append(shares_item)



        slot_id = self.slot_id

        slot_name = self.slot_name

        slot_price_cents = self.slot_price_cents

        status = self.status.value

        campaign: dict[str, Any] | Unset = UNSET
        if not isinstance(self.campaign, Unset):
            campaign = self.campaign.to_dict()

        canceled_at: str | Unset = UNSET
        if not isinstance(self.canceled_at, Unset):
            canceled_at = self.canceled_at.isoformat()

        creative_due: str | Unset = UNSET
        if not isinstance(self.creative_due, Unset):
            creative_due = self.creative_due.isoformat()

        hold_until: str | Unset = UNSET
        if not isinstance(self.hold_until, Unset):
            hold_until = self.hold_until.isoformat()

        proposal: dict[str, Any] | Unset = UNSET
        if not isinstance(self.proposal, Unset):
            proposal = self.proposal.to_dict()

        ready_at: str | Unset = UNSET
        if not isinstance(self.ready_at, Unset):
            ready_at = self.ready_at.isoformat()

        report: dict[str, Any] | Unset = UNSET
        if not isinstance(self.report, Unset):
            report = self.report.to_dict()

        sponsor: dict[str, Any] | Unset = UNSET
        if not isinstance(self.sponsor, Unset):
            sponsor = self.sponsor.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "activity": activity,
            "canManage": can_manage,
            "canShare": can_share,
            "creative": creative,
            "creativeStatus": creative_status,
            "currency": currency,
            "helpers": helpers,
            "id": id,
            "issue": issue,
            "issueOn": issue_on,
            "label": label,
            "library": library,
            "listName": list_name,
            "notes": notes,
            "payment": payment,
            "placement": placement,
            "priceCents": price_cents,
            "shares": shares,
            "slotId": slot_id,
            "slotName": slot_name,
            "slotPriceCents": slot_price_cents,
            "status": status,
        })
        if campaign is not UNSET:
            field_dict["campaign"] = campaign
        if canceled_at is not UNSET:
            field_dict["canceledAt"] = canceled_at
        if creative_due is not UNSET:
            field_dict["creativeDue"] = creative_due
        if hold_until is not UNSET:
            field_dict["holdUntil"] = hold_until
        if proposal is not UNSET:
            field_dict["proposal"] = proposal
        if ready_at is not UNSET:
            field_dict["readyAt"] = ready_at
        if report is not UNSET:
            field_dict["report"] = report
        if sponsor is not UNSET:
            field_dict["sponsor"] = sponsor

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sponsor_booking_event import SponsorBookingEvent # noqa: PLC0415
        from ..models.sponsor_campaign_ref import SponsorCampaignRef # noqa: PLC0415
        from ..models.sponsor_creative import SponsorCreative # noqa: PLC0415
        from ..models.sponsor_item import SponsorItem # noqa: PLC0415
        from ..models.sponsor_payment import SponsorPayment # noqa: PLC0415
        from ..models.sponsor_propose import SponsorPropose # noqa: PLC0415
        from ..models.sponsor_ref import SponsorRef # noqa: PLC0415
        from ..models.sponsor_report import SponsorReport # noqa: PLC0415
        from ..models.sponsor_share_link import SponsorShareLink # noqa: PLC0415
        d = dict(src_dict)
        activity = []
        _activity = d.pop("activity")
        for activity_item_data in (_activity):
            activity_item = SponsorBookingEvent.from_dict(activity_item_data)



            activity.append(activity_item)


        can_manage = d.pop("canManage")

        can_share = d.pop("canShare")

        creative = SponsorCreative.from_dict(d.pop("creative"))




        creative_status = SponsorBookingDetailCreativeStatus(d.pop("creativeStatus"))




        currency = d.pop("currency")

        helpers = cast(list[str], d.pop("helpers"))


        id = d.pop("id")

        issue = d.pop("issue")

        issue_on = datetime.datetime.fromisoformat(d.pop("issueOn"))




        label = d.pop("label")

        library = []
        _library = d.pop("library")
        for library_item_data in (_library):
            library_item = SponsorRef.from_dict(library_item_data)



            library.append(library_item)


        list_name = d.pop("listName")

        notes = d.pop("notes")

        payment = SponsorPayment.from_dict(d.pop("payment"))




        placement = d.pop("placement")

        price_cents = d.pop("priceCents")

        shares = []
        _shares = d.pop("shares")
        for shares_item_data in (_shares):
            shares_item = SponsorShareLink.from_dict(shares_item_data)



            shares.append(shares_item)


        slot_id = d.pop("slotId")

        slot_name = d.pop("slotName")

        slot_price_cents = d.pop("slotPriceCents")

        status = SponsorBookingDetailStatus(d.pop("status"))




        _campaign = d.pop("campaign", UNSET)
        campaign: SponsorCampaignRef | Unset
        if isinstance(_campaign,  Unset):
            campaign = UNSET
        else:
            campaign = SponsorCampaignRef.from_dict(_campaign)




        _canceled_at = d.pop("canceledAt", UNSET)
        canceled_at: datetime.datetime | Unset
        if isinstance(_canceled_at,  Unset):
            canceled_at = UNSET
        else:
            canceled_at = datetime.datetime.fromisoformat(_canceled_at)




        _creative_due = d.pop("creativeDue", UNSET)
        creative_due: datetime.datetime | Unset
        if isinstance(_creative_due,  Unset):
            creative_due = UNSET
        else:
            creative_due = datetime.datetime.fromisoformat(_creative_due)




        _hold_until = d.pop("holdUntil", UNSET)
        hold_until: datetime.datetime | Unset
        if isinstance(_hold_until,  Unset):
            hold_until = UNSET
        else:
            hold_until = datetime.datetime.fromisoformat(_hold_until)




        _proposal = d.pop("proposal", UNSET)
        proposal: SponsorPropose | Unset
        if isinstance(_proposal,  Unset):
            proposal = UNSET
        else:
            proposal = SponsorPropose.from_dict(_proposal)




        _ready_at = d.pop("readyAt", UNSET)
        ready_at: datetime.datetime | Unset
        if isinstance(_ready_at,  Unset):
            ready_at = UNSET
        else:
            ready_at = datetime.datetime.fromisoformat(_ready_at)




        _report = d.pop("report", UNSET)
        report: SponsorReport | Unset
        if isinstance(_report,  Unset):
            report = UNSET
        else:
            report = SponsorReport.from_dict(_report)




        _sponsor = d.pop("sponsor", UNSET)
        sponsor: SponsorItem | Unset
        if isinstance(_sponsor,  Unset):
            sponsor = UNSET
        else:
            sponsor = SponsorItem.from_dict(_sponsor)




        sponsor_booking_detail = cls(
            activity=activity,
            can_manage=can_manage,
            can_share=can_share,
            creative=creative,
            creative_status=creative_status,
            currency=currency,
            helpers=helpers,
            id=id,
            issue=issue,
            issue_on=issue_on,
            label=label,
            library=library,
            list_name=list_name,
            notes=notes,
            payment=payment,
            placement=placement,
            price_cents=price_cents,
            shares=shares,
            slot_id=slot_id,
            slot_name=slot_name,
            slot_price_cents=slot_price_cents,
            status=status,
            campaign=campaign,
            canceled_at=canceled_at,
            creative_due=creative_due,
            hold_until=hold_until,
            proposal=proposal,
            ready_at=ready_at,
            report=report,
            sponsor=sponsor,
        )

        return sponsor_booking_detail

