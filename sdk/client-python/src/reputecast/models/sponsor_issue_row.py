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
  from ..models.sponsor_booking_brief import SponsorBookingBrief
  from ..models.sponsor_campaign_ref import SponsorCampaignRef
  from ..models.sponsor_propose import SponsorPropose





T = TypeVar("T", bound="SponsorIssueRow")



@_attrs_define
class SponsorIssueRow:
    """ 
        Attributes:
            issue_on (datetime.datetime):
            number (int):
            past (bool):
            slot_id (str):
            slot_name (str):
            booking (SponsorBookingBrief | Unset):
            campaign (SponsorCampaignRef | Unset):
            proposal (SponsorPropose | Unset):
     """

    issue_on: datetime.datetime
    number: int
    past: bool
    slot_id: str
    slot_name: str
    booking: SponsorBookingBrief | Unset = UNSET
    campaign: SponsorCampaignRef | Unset = UNSET
    proposal: SponsorPropose | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sponsor_booking_brief import SponsorBookingBrief # noqa: PLC0415
        from ..models.sponsor_campaign_ref import SponsorCampaignRef # noqa: PLC0415
        from ..models.sponsor_propose import SponsorPropose # noqa: PLC0415
        issue_on = self.issue_on.isoformat()

        number = self.number

        past = self.past

        slot_id = self.slot_id

        slot_name = self.slot_name

        booking: dict[str, Any] | Unset = UNSET
        if not isinstance(self.booking, Unset):
            booking = self.booking.to_dict()

        campaign: dict[str, Any] | Unset = UNSET
        if not isinstance(self.campaign, Unset):
            campaign = self.campaign.to_dict()

        proposal: dict[str, Any] | Unset = UNSET
        if not isinstance(self.proposal, Unset):
            proposal = self.proposal.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "issueOn": issue_on,
            "number": number,
            "past": past,
            "slotId": slot_id,
            "slotName": slot_name,
        })
        if booking is not UNSET:
            field_dict["booking"] = booking
        if campaign is not UNSET:
            field_dict["campaign"] = campaign
        if proposal is not UNSET:
            field_dict["proposal"] = proposal

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sponsor_booking_brief import SponsorBookingBrief # noqa: PLC0415
        from ..models.sponsor_campaign_ref import SponsorCampaignRef # noqa: PLC0415
        from ..models.sponsor_propose import SponsorPropose # noqa: PLC0415
        d = dict(src_dict)
        issue_on = datetime.datetime.fromisoformat(d.pop("issueOn"))




        number = d.pop("number")

        past = d.pop("past")

        slot_id = d.pop("slotId")

        slot_name = d.pop("slotName")

        _booking = d.pop("booking", UNSET)
        booking: SponsorBookingBrief | Unset
        if isinstance(_booking,  Unset):
            booking = UNSET
        else:
            booking = SponsorBookingBrief.from_dict(_booking)




        _campaign = d.pop("campaign", UNSET)
        campaign: SponsorCampaignRef | Unset
        if isinstance(_campaign,  Unset):
            campaign = UNSET
        else:
            campaign = SponsorCampaignRef.from_dict(_campaign)




        _proposal = d.pop("proposal", UNSET)
        proposal: SponsorPropose | Unset
        if isinstance(_proposal,  Unset):
            proposal = UNSET
        else:
            proposal = SponsorPropose.from_dict(_proposal)




        sponsor_issue_row = cls(
            issue_on=issue_on,
            number=number,
            past=past,
            slot_id=slot_id,
            slot_name=slot_name,
            booking=booking,
            campaign=campaign,
            proposal=proposal,
        )

        return sponsor_issue_row

