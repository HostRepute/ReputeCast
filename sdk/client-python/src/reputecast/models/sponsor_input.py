from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="SponsorInput")



@_attrs_define
class SponsorInput:
    """ 
        Attributes:
            name (str):
            contact_email (str | Unset):
            contact_name (str | Unset):
            logo_url (str | Unset):
            notes (str | Unset):
            terms_days (int | Unset): Payments are due this many days after the issue (14 when absent).
            website (str | Unset):
     """

    name: str
    contact_email: str | Unset = UNSET
    contact_name: str | Unset = UNSET
    logo_url: str | Unset = UNSET
    notes: str | Unset = UNSET
    terms_days: int | Unset = UNSET
    website: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        contact_email = self.contact_email

        contact_name = self.contact_name

        logo_url = self.logo_url

        notes = self.notes

        terms_days = self.terms_days

        website = self.website


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
        })
        if contact_email is not UNSET:
            field_dict["contactEmail"] = contact_email
        if contact_name is not UNSET:
            field_dict["contactName"] = contact_name
        if logo_url is not UNSET:
            field_dict["logoUrl"] = logo_url
        if notes is not UNSET:
            field_dict["notes"] = notes
        if terms_days is not UNSET:
            field_dict["termsDays"] = terms_days
        if website is not UNSET:
            field_dict["website"] = website

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        contact_email = d.pop("contactEmail", UNSET)

        contact_name = d.pop("contactName", UNSET)

        logo_url = d.pop("logoUrl", UNSET)

        notes = d.pop("notes", UNSET)

        terms_days = d.pop("termsDays", UNSET)

        website = d.pop("website", UNSET)

        sponsor_input = cls(
            name=name,
            contact_email=contact_email,
            contact_name=contact_name,
            logo_url=logo_url,
            notes=notes,
            terms_days=terms_days,
            website=website,
        )

        return sponsor_input

