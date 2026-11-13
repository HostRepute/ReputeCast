from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sponsor_booking_brief_creative_status import SponsorBookingBriefCreativeStatus
from ..models.sponsor_booking_brief_status import SponsorBookingBriefStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.sponsor_payment import SponsorPayment
  from ..models.sponsor_propose import SponsorPropose
  from ..models.sponsor_ref import SponsorRef





T = TypeVar("T", bound="SponsorBookingBrief")



@_attrs_define
class SponsorBookingBrief:
    """ 
        Attributes:
            creative_status (SponsorBookingBriefCreativeStatus):
            final (bool):
            id (str):
            payment (SponsorPayment):
            price_cents (int):
            status (SponsorBookingBriefStatus):
            click_rate (float | Unset):
            creative_due (datetime.datetime | Unset):
            hold_until (datetime.datetime | Unset):
            payment_due_at (datetime.datetime | Unset):
            proposal (SponsorPropose | Unset):
            results_final_at (datetime.datetime | Unset):
            sent_at (datetime.datetime | Unset):
            shared_at (datetime.datetime | Unset):
            sponsor (SponsorRef | Unset):
     """

    creative_status: SponsorBookingBriefCreativeStatus
    final: bool
    id: str
    payment: SponsorPayment
    price_cents: int
    status: SponsorBookingBriefStatus
    click_rate: float | Unset = UNSET
    creative_due: datetime.datetime | Unset = UNSET
    hold_until: datetime.datetime | Unset = UNSET
    payment_due_at: datetime.datetime | Unset = UNSET
    proposal: SponsorPropose | Unset = UNSET
    results_final_at: datetime.datetime | Unset = UNSET
    sent_at: datetime.datetime | Unset = UNSET
    shared_at: datetime.datetime | Unset = UNSET
    sponsor: SponsorRef | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sponsor_payment import SponsorPayment # noqa: PLC0415
        from ..models.sponsor_propose import SponsorPropose # noqa: PLC0415
        from ..models.sponsor_ref import SponsorRef # noqa: PLC0415
        creative_status = self.creative_status.value

        final = self.final

        id = self.id

        payment = self.payment.to_dict()

        price_cents = self.price_cents

        status = self.status.value

        click_rate = self.click_rate

        creative_due: str | Unset = UNSET
        if not isinstance(self.creative_due, Unset):
            creative_due = self.creative_due.isoformat()

        hold_until: str | Unset = UNSET
        if not isinstance(self.hold_until, Unset):
            hold_until = self.hold_until.isoformat()

        payment_due_at: str | Unset = UNSET
        if not isinstance(self.payment_due_at, Unset):
            payment_due_at = self.payment_due_at.isoformat()

        proposal: dict[str, Any] | Unset = UNSET
        if not isinstance(self.proposal, Unset):
            proposal = self.proposal.to_dict()

        results_final_at: str | Unset = UNSET
        if not isinstance(self.results_final_at, Unset):
            results_final_at = self.results_final_at.isoformat()

        sent_at: str | Unset = UNSET
        if not isinstance(self.sent_at, Unset):
            sent_at = self.sent_at.isoformat()

        shared_at: str | Unset = UNSET
        if not isinstance(self.shared_at, Unset):
            shared_at = self.shared_at.isoformat()

        sponsor: dict[str, Any] | Unset = UNSET
        if not isinstance(self.sponsor, Unset):
            sponsor = self.sponsor.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "creativeStatus": creative_status,
            "final": final,
            "id": id,
            "payment": payment,
            "priceCents": price_cents,
            "status": status,
        })
        if click_rate is not UNSET:
            field_dict["clickRate"] = click_rate
        if creative_due is not UNSET:
            field_dict["creativeDue"] = creative_due
        if hold_until is not UNSET:
            field_dict["holdUntil"] = hold_until
        if payment_due_at is not UNSET:
            field_dict["paymentDueAt"] = payment_due_at
        if proposal is not UNSET:
            field_dict["proposal"] = proposal
        if results_final_at is not UNSET:
            field_dict["resultsFinalAt"] = results_final_at
        if sent_at is not UNSET:
            field_dict["sentAt"] = sent_at
        if shared_at is not UNSET:
            field_dict["sharedAt"] = shared_at
        if sponsor is not UNSET:
            field_dict["sponsor"] = sponsor

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sponsor_payment import SponsorPayment # noqa: PLC0415
        from ..models.sponsor_propose import SponsorPropose # noqa: PLC0415
        from ..models.sponsor_ref import SponsorRef # noqa: PLC0415
        d = dict(src_dict)
        creative_status = SponsorBookingBriefCreativeStatus(d.pop("creativeStatus"))




        final = d.pop("final")

        id = d.pop("id")

        payment = SponsorPayment.from_dict(d.pop("payment"))




        price_cents = d.pop("priceCents")

        status = SponsorBookingBriefStatus(d.pop("status"))




        click_rate = d.pop("clickRate", UNSET)

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




        _payment_due_at = d.pop("paymentDueAt", UNSET)
        payment_due_at: datetime.datetime | Unset
        if isinstance(_payment_due_at,  Unset):
            payment_due_at = UNSET
        else:
            payment_due_at = datetime.datetime.fromisoformat(_payment_due_at)




        _proposal = d.pop("proposal", UNSET)
        proposal: SponsorPropose | Unset
        if isinstance(_proposal,  Unset):
            proposal = UNSET
        else:
            proposal = SponsorPropose.from_dict(_proposal)




        _results_final_at = d.pop("resultsFinalAt", UNSET)
        results_final_at: datetime.datetime | Unset
        if isinstance(_results_final_at,  Unset):
            results_final_at = UNSET
        else:
            results_final_at = datetime.datetime.fromisoformat(_results_final_at)




        _sent_at = d.pop("sentAt", UNSET)
        sent_at: datetime.datetime | Unset
        if isinstance(_sent_at,  Unset):
            sent_at = UNSET
        else:
            sent_at = datetime.datetime.fromisoformat(_sent_at)




        _shared_at = d.pop("sharedAt", UNSET)
        shared_at: datetime.datetime | Unset
        if isinstance(_shared_at,  Unset):
            shared_at = UNSET
        else:
            shared_at = datetime.datetime.fromisoformat(_shared_at)




        _sponsor = d.pop("sponsor", UNSET)
        sponsor: SponsorRef | Unset
        if isinstance(_sponsor,  Unset):
            sponsor = UNSET
        else:
            sponsor = SponsorRef.from_dict(_sponsor)




        sponsor_booking_brief = cls(
            creative_status=creative_status,
            final=final,
            id=id,
            payment=payment,
            price_cents=price_cents,
            status=status,
            click_rate=click_rate,
            creative_due=creative_due,
            hold_until=hold_until,
            payment_due_at=payment_due_at,
            proposal=proposal,
            results_final_at=results_final_at,
            sent_at=sent_at,
            shared_at=shared_at,
            sponsor=sponsor,
        )

        return sponsor_booking_brief

