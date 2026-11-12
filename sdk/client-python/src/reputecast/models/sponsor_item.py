from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="SponsorItem")



@_attrs_define
class SponsorItem:
    """ 
        Attributes:
            booked_cents (int):
            contact_email (str):
            contact_name (str):
            currency (str):
            hold (list[int]):
            hold_cents (int):
            id (str):
            initials (str):
            issues (list[int]):
            logo_url (str):
            name (str):
            notes (str):
            paid_cents (int):
            terms_days (int):
            website (str):
            click_rate (float | Unset):
            click_rate_issue (int | Unset):
     """

    booked_cents: int
    contact_email: str
    contact_name: str
    currency: str
    hold: list[int]
    hold_cents: int
    id: str
    initials: str
    issues: list[int]
    logo_url: str
    name: str
    notes: str
    paid_cents: int
    terms_days: int
    website: str
    click_rate: float | Unset = UNSET
    click_rate_issue: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        booked_cents = self.booked_cents

        contact_email = self.contact_email

        contact_name = self.contact_name

        currency = self.currency

        hold = self.hold



        hold_cents = self.hold_cents

        id = self.id

        initials = self.initials

        issues = self.issues



        logo_url = self.logo_url

        name = self.name

        notes = self.notes

        paid_cents = self.paid_cents

        terms_days = self.terms_days

        website = self.website

        click_rate = self.click_rate

        click_rate_issue = self.click_rate_issue


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bookedCents": booked_cents,
            "contactEmail": contact_email,
            "contactName": contact_name,
            "currency": currency,
            "hold": hold,
            "holdCents": hold_cents,
            "id": id,
            "initials": initials,
            "issues": issues,
            "logoUrl": logo_url,
            "name": name,
            "notes": notes,
            "paidCents": paid_cents,
            "termsDays": terms_days,
            "website": website,
        })
        if click_rate is not UNSET:
            field_dict["clickRate"] = click_rate
        if click_rate_issue is not UNSET:
            field_dict["clickRateIssue"] = click_rate_issue

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        booked_cents = d.pop("bookedCents")

        contact_email = d.pop("contactEmail")

        contact_name = d.pop("contactName")

        currency = d.pop("currency")

        hold = cast(list[int], d.pop("hold"))


        hold_cents = d.pop("holdCents")

        id = d.pop("id")

        initials = d.pop("initials")

        issues = cast(list[int], d.pop("issues"))


        logo_url = d.pop("logoUrl")

        name = d.pop("name")

        notes = d.pop("notes")

        paid_cents = d.pop("paidCents")

        terms_days = d.pop("termsDays")

        website = d.pop("website")

        click_rate = d.pop("clickRate", UNSET)

        click_rate_issue = d.pop("clickRateIssue", UNSET)

        sponsor_item = cls(
            booked_cents=booked_cents,
            contact_email=contact_email,
            contact_name=contact_name,
            currency=currency,
            hold=hold,
            hold_cents=hold_cents,
            id=id,
            initials=initials,
            issues=issues,
            logo_url=logo_url,
            name=name,
            notes=notes,
            paid_cents=paid_cents,
            terms_days=terms_days,
            website=website,
            click_rate=click_rate,
            click_rate_issue=click_rate_issue,
        )

        return sponsor_item

