from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="EarnImportStats")



@_attrs_define
class EarnImportStats:
    """ 
        Attributes:
            complimentary (int):
            emails_sent (int):
            free (int):
            merged (int):
            new (int):
            paid (int):
            prices_mapped (int):
            skipped (int):
            comp_first (datetime.datetime | Unset):
            comp_last (datetime.datetime | Unset):
     """

    complimentary: int
    emails_sent: int
    free: int
    merged: int
    new: int
    paid: int
    prices_mapped: int
    skipped: int
    comp_first: datetime.datetime | Unset = UNSET
    comp_last: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        complimentary = self.complimentary

        emails_sent = self.emails_sent

        free = self.free

        merged = self.merged

        new = self.new

        paid = self.paid

        prices_mapped = self.prices_mapped

        skipped = self.skipped

        comp_first: str | Unset = UNSET
        if not isinstance(self.comp_first, Unset):
            comp_first = self.comp_first.isoformat()

        comp_last: str | Unset = UNSET
        if not isinstance(self.comp_last, Unset):
            comp_last = self.comp_last.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "complimentary": complimentary,
            "emailsSent": emails_sent,
            "free": free,
            "merged": merged,
            "new": new,
            "paid": paid,
            "pricesMapped": prices_mapped,
            "skipped": skipped,
        })
        if comp_first is not UNSET:
            field_dict["compFirst"] = comp_first
        if comp_last is not UNSET:
            field_dict["compLast"] = comp_last

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        complimentary = d.pop("complimentary")

        emails_sent = d.pop("emailsSent")

        free = d.pop("free")

        merged = d.pop("merged")

        new = d.pop("new")

        paid = d.pop("paid")

        prices_mapped = d.pop("pricesMapped")

        skipped = d.pop("skipped")

        _comp_first = d.pop("compFirst", UNSET)
        comp_first: datetime.datetime | Unset
        if isinstance(_comp_first,  Unset):
            comp_first = UNSET
        else:
            comp_first = datetime.datetime.fromisoformat(_comp_first)




        _comp_last = d.pop("compLast", UNSET)
        comp_last: datetime.datetime | Unset
        if isinstance(_comp_last,  Unset):
            comp_last = UNSET
        else:
            comp_last = datetime.datetime.fromisoformat(_comp_last)




        earn_import_stats = cls(
            complimentary=complimentary,
            emails_sent=emails_sent,
            free=free,
            merged=merged,
            new=new,
            paid=paid,
            prices_mapped=prices_mapped,
            skipped=skipped,
            comp_first=comp_first,
            comp_last=comp_last,
        )

        return earn_import_stats

