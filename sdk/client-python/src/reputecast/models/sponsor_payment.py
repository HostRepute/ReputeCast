from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sponsor_payment_status import SponsorPaymentStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SponsorPayment")



@_attrs_define
class SponsorPayment:
    """ 
        Attributes:
            note (str):
            status (SponsorPaymentStatus):
            due (datetime.datetime | Unset):
            paid_cents (int | Unset):
            paid_on (datetime.datetime | Unset):
            recorded_at (datetime.datetime | Unset):
            recorded_by (str | Unset):
     """

    note: str
    status: SponsorPaymentStatus
    due: datetime.datetime | Unset = UNSET
    paid_cents: int | Unset = UNSET
    paid_on: datetime.datetime | Unset = UNSET
    recorded_at: datetime.datetime | Unset = UNSET
    recorded_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        note = self.note

        status = self.status.value

        due: str | Unset = UNSET
        if not isinstance(self.due, Unset):
            due = self.due.isoformat()

        paid_cents = self.paid_cents

        paid_on: str | Unset = UNSET
        if not isinstance(self.paid_on, Unset):
            paid_on = self.paid_on.isoformat()

        recorded_at: str | Unset = UNSET
        if not isinstance(self.recorded_at, Unset):
            recorded_at = self.recorded_at.isoformat()

        recorded_by = self.recorded_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "note": note,
            "status": status,
        })
        if due is not UNSET:
            field_dict["due"] = due
        if paid_cents is not UNSET:
            field_dict["paidCents"] = paid_cents
        if paid_on is not UNSET:
            field_dict["paidOn"] = paid_on
        if recorded_at is not UNSET:
            field_dict["recordedAt"] = recorded_at
        if recorded_by is not UNSET:
            field_dict["recordedBy"] = recorded_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        note = d.pop("note")

        status = SponsorPaymentStatus(d.pop("status"))




        _due = d.pop("due", UNSET)
        due: datetime.datetime | Unset
        if isinstance(_due,  Unset):
            due = UNSET
        else:
            due = datetime.datetime.fromisoformat(_due)




        paid_cents = d.pop("paidCents", UNSET)

        _paid_on = d.pop("paidOn", UNSET)
        paid_on: datetime.datetime | Unset
        if isinstance(_paid_on,  Unset):
            paid_on = UNSET
        else:
            paid_on = datetime.datetime.fromisoformat(_paid_on)




        _recorded_at = d.pop("recordedAt", UNSET)
        recorded_at: datetime.datetime | Unset
        if isinstance(_recorded_at,  Unset):
            recorded_at = UNSET
        else:
            recorded_at = datetime.datetime.fromisoformat(_recorded_at)




        recorded_by = d.pop("recordedBy", UNSET)

        sponsor_payment = cls(
            note=note,
            status=status,
            due=due,
            paid_cents=paid_cents,
            paid_on=paid_on,
            recorded_at=recorded_at,
            recorded_by=recorded_by,
        )

        return sponsor_payment

