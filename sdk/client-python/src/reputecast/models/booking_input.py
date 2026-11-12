from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.booking_input_status import BookingInputStatus
from ..types import UNSET, Unset
from typing import cast
from uuid import UUID
import datetime






T = TypeVar("T", bound="BookingInput")



@_attrs_define
class BookingInput:
    """ 
        Attributes:
            hold_until (datetime.date | Unset):
            notes (str | Unset):
            price_cents (int | Unset):
            sponsor_id (UUID | Unset):
            status (BookingInputStatus | Unset):
     """

    hold_until: datetime.date | Unset = UNSET
    notes: str | Unset = UNSET
    price_cents: int | Unset = UNSET
    sponsor_id: UUID | Unset = UNSET
    status: BookingInputStatus | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        hold_until: str | Unset = UNSET
        if not isinstance(self.hold_until, Unset):
            hold_until = self.hold_until.isoformat()

        notes = self.notes

        price_cents = self.price_cents

        sponsor_id: str | Unset = UNSET
        if not isinstance(self.sponsor_id, Unset):
            sponsor_id = str(self.sponsor_id)

        status: str | Unset = UNSET
        if not isinstance(self.status, Unset):
            status = self.status.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if hold_until is not UNSET:
            field_dict["holdUntil"] = hold_until
        if notes is not UNSET:
            field_dict["notes"] = notes
        if price_cents is not UNSET:
            field_dict["priceCents"] = price_cents
        if sponsor_id is not UNSET:
            field_dict["sponsorId"] = sponsor_id
        if status is not UNSET:
            field_dict["status"] = status

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        _hold_until = d.pop("holdUntil", UNSET)
        hold_until: datetime.date | Unset
        if isinstance(_hold_until,  Unset):
            hold_until = UNSET
        else:
            hold_until = datetime.date.fromisoformat(_hold_until)




        notes = d.pop("notes", UNSET)

        price_cents = d.pop("priceCents", UNSET)

        _sponsor_id = d.pop("sponsorId", UNSET)
        sponsor_id: UUID | Unset
        if isinstance(_sponsor_id,  Unset):
            sponsor_id = UNSET
        else:
            sponsor_id = UUID(_sponsor_id)




        _status = d.pop("status", UNSET)
        status: BookingInputStatus | Unset
        if isinstance(_status,  Unset):
            status = UNSET
        else:
            status = BookingInputStatus(_status)




        booking_input = cls(
            hold_until=hold_until,
            notes=notes,
            price_cents=price_cents,
            sponsor_id=sponsor_id,
            status=status,
        )

        return booking_input

