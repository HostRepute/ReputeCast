from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SponsorItem")



@_attrs_define
class SponsorItem:
    """ 
        Attributes:
            awaiting (list[int]):
            awaiting_cents (int):
            booked_cents (int):
            contact_email (str):
            contact_name (str):
            currency (str):
            from_request (list[int]):
            hold (list[int]):
            hold_cents (int):
            id (str):
            initials (str):
            issues (list[int]):
            logo_url (str):
            name (str):
            notes (str):
            paid_cents (int):
            requests (int):
            terms_days (int):
            website (str):
            awaiting_due (datetime.datetime | Unset):
            click_rate (float | Unset):
            click_rate_issue (int | Unset):
            stripe_paid_on (datetime.datetime | Unset):
     """

    awaiting: list[int]
    awaiting_cents: int
    booked_cents: int
    contact_email: str
    contact_name: str
    currency: str
    from_request: list[int]
    hold: list[int]
    hold_cents: int
    id: str
    initials: str
    issues: list[int]
    logo_url: str
    name: str
    notes: str
    paid_cents: int
    requests: int
    terms_days: int
    website: str
    awaiting_due: datetime.datetime | Unset = UNSET
    click_rate: float | Unset = UNSET
    click_rate_issue: int | Unset = UNSET
    stripe_paid_on: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        awaiting = self.awaiting



        awaiting_cents = self.awaiting_cents

        booked_cents = self.booked_cents

        contact_email = self.contact_email

        contact_name = self.contact_name

        currency = self.currency

        from_request = self.from_request



        hold = self.hold



        hold_cents = self.hold_cents

        id = self.id

        initials = self.initials

        issues = self.issues



        logo_url = self.logo_url

        name = self.name

        notes = self.notes

        paid_cents = self.paid_cents

        requests = self.requests

        terms_days = self.terms_days

        website = self.website

        awaiting_due: str | Unset = UNSET
        if not isinstance(self.awaiting_due, Unset):
            awaiting_due = self.awaiting_due.isoformat()

        click_rate = self.click_rate

        click_rate_issue = self.click_rate_issue

        stripe_paid_on: str | Unset = UNSET
        if not isinstance(self.stripe_paid_on, Unset):
            stripe_paid_on = self.stripe_paid_on.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "awaiting": awaiting,
            "awaitingCents": awaiting_cents,
            "bookedCents": booked_cents,
            "contactEmail": contact_email,
            "contactName": contact_name,
            "currency": currency,
            "fromRequest": from_request,
            "hold": hold,
            "holdCents": hold_cents,
            "id": id,
            "initials": initials,
            "issues": issues,
            "logoUrl": logo_url,
            "name": name,
            "notes": notes,
            "paidCents": paid_cents,
            "requests": requests,
            "termsDays": terms_days,
            "website": website,
        })
        if awaiting_due is not UNSET:
            field_dict["awaitingDue"] = awaiting_due
        if click_rate is not UNSET:
            field_dict["clickRate"] = click_rate
        if click_rate_issue is not UNSET:
            field_dict["clickRateIssue"] = click_rate_issue
        if stripe_paid_on is not UNSET:
            field_dict["stripePaidOn"] = stripe_paid_on

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        awaiting = cast(list[int], d.pop("awaiting"))


        awaiting_cents = d.pop("awaitingCents")

        booked_cents = d.pop("bookedCents")

        contact_email = d.pop("contactEmail")

        contact_name = d.pop("contactName")

        currency = d.pop("currency")

        from_request = cast(list[int], d.pop("fromRequest"))


        hold = cast(list[int], d.pop("hold"))


        hold_cents = d.pop("holdCents")

        id = d.pop("id")

        initials = d.pop("initials")

        issues = cast(list[int], d.pop("issues"))


        logo_url = d.pop("logoUrl")

        name = d.pop("name")

        notes = d.pop("notes")

        paid_cents = d.pop("paidCents")

        requests = d.pop("requests")

        terms_days = d.pop("termsDays")

        website = d.pop("website")

        _awaiting_due = d.pop("awaitingDue", UNSET)
        awaiting_due: datetime.datetime | Unset
        if isinstance(_awaiting_due,  Unset):
            awaiting_due = UNSET
        else:
            awaiting_due = datetime.datetime.fromisoformat(_awaiting_due)




        click_rate = d.pop("clickRate", UNSET)

        click_rate_issue = d.pop("clickRateIssue", UNSET)

        _stripe_paid_on = d.pop("stripePaidOn", UNSET)
        stripe_paid_on: datetime.datetime | Unset
        if isinstance(_stripe_paid_on,  Unset):
            stripe_paid_on = UNSET
        else:
            stripe_paid_on = datetime.datetime.fromisoformat(_stripe_paid_on)




        sponsor_item = cls(
            awaiting=awaiting,
            awaiting_cents=awaiting_cents,
            booked_cents=booked_cents,
            contact_email=contact_email,
            contact_name=contact_name,
            currency=currency,
            from_request=from_request,
            hold=hold,
            hold_cents=hold_cents,
            id=id,
            initials=initials,
            issues=issues,
            logo_url=logo_url,
            name=name,
            notes=notes,
            paid_cents=paid_cents,
            requests=requests,
            terms_days=terms_days,
            website=website,
            awaiting_due=awaiting_due,
            click_rate=click_rate,
            click_rate_issue=click_rate_issue,
            stripe_paid_on=stripe_paid_on,
        )

        return sponsor_item

