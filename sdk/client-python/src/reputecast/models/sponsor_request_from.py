from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SponsorRequestFrom")



@_attrs_define
class SponsorRequestFrom:
    """ 
        Attributes:
            at (datetime.datetime):
            company (str):
            email (str):
            name (str):
            notes (str):
            website (str):
            approved_at (datetime.datetime | Unset):
            approved_by (str | Unset):
     """

    at: datetime.datetime
    company: str
    email: str
    name: str
    notes: str
    website: str
    approved_at: datetime.datetime | Unset = UNSET
    approved_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        company = self.company

        email = self.email

        name = self.name

        notes = self.notes

        website = self.website

        approved_at: str | Unset = UNSET
        if not isinstance(self.approved_at, Unset):
            approved_at = self.approved_at.isoformat()

        approved_by = self.approved_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "company": company,
            "email": email,
            "name": name,
            "notes": notes,
            "website": website,
        })
        if approved_at is not UNSET:
            field_dict["approvedAt"] = approved_at
        if approved_by is not UNSET:
            field_dict["approvedBy"] = approved_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        company = d.pop("company")

        email = d.pop("email")

        name = d.pop("name")

        notes = d.pop("notes")

        website = d.pop("website")

        _approved_at = d.pop("approvedAt", UNSET)
        approved_at: datetime.datetime | Unset
        if isinstance(_approved_at,  Unset):
            approved_at = UNSET
        else:
            approved_at = datetime.datetime.fromisoformat(_approved_at)




        approved_by = d.pop("approvedBy", UNSET)

        sponsor_request_from = cls(
            at=at,
            company=company,
            email=email,
            name=name,
            notes=notes,
            website=website,
            approved_at=approved_at,
            approved_by=approved_by,
        )

        return sponsor_request_from

