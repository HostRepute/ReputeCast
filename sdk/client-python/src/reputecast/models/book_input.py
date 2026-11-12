from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.book_input_status import BookInputStatus
from ..types import UNSET, Unset
from typing import cast
from uuid import UUID
import datetime






T = TypeVar("T", bound="BookInput")



@_attrs_define
class BookInput:
    """ 
        Attributes:
            sponsor_id (UUID):
            status (BookInputStatus):
            hold_until (datetime.date | Unset):
            notes (str | Unset):
            price_cents (int | Unset): The slot's default price when absent.
     """

    sponsor_id: UUID
    status: BookInputStatus
    hold_until: datetime.date | Unset = UNSET
    notes: str | Unset = UNSET
    price_cents: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        sponsor_id = str(self.sponsor_id)

        status = self.status.value

        hold_until: str | Unset = UNSET
        if not isinstance(self.hold_until, Unset):
            hold_until = self.hold_until.isoformat()

        notes = self.notes

        price_cents = self.price_cents


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "sponsorId": sponsor_id,
            "status": status,
        })
        if hold_until is not UNSET:
            field_dict["holdUntil"] = hold_until
        if notes is not UNSET:
            field_dict["notes"] = notes
        if price_cents is not UNSET:
            field_dict["priceCents"] = price_cents

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        sponsor_id = UUID(d.pop("sponsorId"))




        status = BookInputStatus(d.pop("status"))




        _hold_until = d.pop("holdUntil", UNSET)
        hold_until: datetime.date | Unset
        if isinstance(_hold_until,  Unset):
            hold_until = UNSET
        else:
            hold_until = datetime.date.fromisoformat(_hold_until)




        notes = d.pop("notes", UNSET)

        price_cents = d.pop("priceCents", UNSET)

        book_input = cls(
            sponsor_id=sponsor_id,
            status=status,
            hold_until=hold_until,
            notes=notes,
            price_cents=price_cents,
        )

        return book_input

