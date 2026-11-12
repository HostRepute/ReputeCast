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
  from ..models.sponsor_campaign_ref import SponsorCampaignRef
  from ..models.sponsor_propose import SponsorPropose
  from ..models.sponsor_ref import SponsorRef





T = TypeVar("T", bound="SponsorIssueDetail")



@_attrs_define
class SponsorIssueDetail:
    """ 
        Attributes:
            can_manage (bool):
            creative_due (datetime.datetime):
            currency (str):
            helpers (list[str]):
            issue (int):
            issue_on (datetime.datetime):
            library (list[SponsorRef]):
            list_name (str):
            payment_due (datetime.datetime):
            placement (str):
            price_cents (int):
            proposals (list[SponsorPropose]):
            slot_id (str):
            slot_name (str):
            terms_days (int):
            booking_id (str | Unset):
            campaign (SponsorCampaignRef | Unset):
     """

    can_manage: bool
    creative_due: datetime.datetime
    currency: str
    helpers: list[str]
    issue: int
    issue_on: datetime.datetime
    library: list[SponsorRef]
    list_name: str
    payment_due: datetime.datetime
    placement: str
    price_cents: int
    proposals: list[SponsorPropose]
    slot_id: str
    slot_name: str
    terms_days: int
    booking_id: str | Unset = UNSET
    campaign: SponsorCampaignRef | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sponsor_campaign_ref import SponsorCampaignRef # noqa: PLC0415
        from ..models.sponsor_propose import SponsorPropose # noqa: PLC0415
        from ..models.sponsor_ref import SponsorRef # noqa: PLC0415
        can_manage = self.can_manage

        creative_due = self.creative_due.isoformat()

        currency = self.currency

        helpers = self.helpers



        issue = self.issue

        issue_on = self.issue_on.isoformat()

        library = []
        for library_item_data in self.library:
            library_item = library_item_data.to_dict()
            library.append(library_item)



        list_name = self.list_name

        payment_due = self.payment_due.isoformat()

        placement = self.placement

        price_cents = self.price_cents

        proposals = []
        for proposals_item_data in self.proposals:
            proposals_item = proposals_item_data.to_dict()
            proposals.append(proposals_item)



        slot_id = self.slot_id

        slot_name = self.slot_name

        terms_days = self.terms_days

        booking_id = self.booking_id

        campaign: dict[str, Any] | Unset = UNSET
        if not isinstance(self.campaign, Unset):
            campaign = self.campaign.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canManage": can_manage,
            "creativeDue": creative_due,
            "currency": currency,
            "helpers": helpers,
            "issue": issue,
            "issueOn": issue_on,
            "library": library,
            "listName": list_name,
            "paymentDue": payment_due,
            "placement": placement,
            "priceCents": price_cents,
            "proposals": proposals,
            "slotId": slot_id,
            "slotName": slot_name,
            "termsDays": terms_days,
        })
        if booking_id is not UNSET:
            field_dict["bookingId"] = booking_id
        if campaign is not UNSET:
            field_dict["campaign"] = campaign

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sponsor_campaign_ref import SponsorCampaignRef # noqa: PLC0415
        from ..models.sponsor_propose import SponsorPropose # noqa: PLC0415
        from ..models.sponsor_ref import SponsorRef # noqa: PLC0415
        d = dict(src_dict)
        can_manage = d.pop("canManage")

        creative_due = datetime.datetime.fromisoformat(d.pop("creativeDue"))




        currency = d.pop("currency")

        helpers = cast(list[str], d.pop("helpers"))


        issue = d.pop("issue")

        issue_on = datetime.datetime.fromisoformat(d.pop("issueOn"))




        library = []
        _library = d.pop("library")
        for library_item_data in (_library):
            library_item = SponsorRef.from_dict(library_item_data)



            library.append(library_item)


        list_name = d.pop("listName")

        payment_due = datetime.datetime.fromisoformat(d.pop("paymentDue"))




        placement = d.pop("placement")

        price_cents = d.pop("priceCents")

        proposals = []
        _proposals = d.pop("proposals")
        for proposals_item_data in (_proposals):
            proposals_item = SponsorPropose.from_dict(proposals_item_data)



            proposals.append(proposals_item)


        slot_id = d.pop("slotId")

        slot_name = d.pop("slotName")

        terms_days = d.pop("termsDays")

        booking_id = d.pop("bookingId", UNSET)

        _campaign = d.pop("campaign", UNSET)
        campaign: SponsorCampaignRef | Unset
        if isinstance(_campaign,  Unset):
            campaign = UNSET
        else:
            campaign = SponsorCampaignRef.from_dict(_campaign)




        sponsor_issue_detail = cls(
            can_manage=can_manage,
            creative_due=creative_due,
            currency=currency,
            helpers=helpers,
            issue=issue,
            issue_on=issue_on,
            library=library,
            list_name=list_name,
            payment_due=payment_due,
            placement=placement,
            price_cents=price_cents,
            proposals=proposals,
            slot_id=slot_id,
            slot_name=slot_name,
            terms_days=terms_days,
            booking_id=booking_id,
            campaign=campaign,
        )

        return sponsor_issue_detail

