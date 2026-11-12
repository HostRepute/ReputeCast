from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.partner_request_item_origin import PartnerRequestItemOrigin
from ..models.partner_request_item_status import PartnerRequestItemStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="PartnerRequestItem")



@_attrs_define
class PartnerRequestItem:
    """ 
        Attributes:
            at (datetime.datetime):
            attempts (int):
            id (str):
            last_error (str):
            origin (PartnerRequestItemOrigin):
            status (PartnerRequestItemStatus):
            confirmed_at (datetime.datetime | Unset):
     """

    at: datetime.datetime
    attempts: int
    id: str
    last_error: str
    origin: PartnerRequestItemOrigin
    status: PartnerRequestItemStatus
    confirmed_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        attempts = self.attempts

        id = self.id

        last_error = self.last_error

        origin = self.origin.value

        status = self.status.value

        confirmed_at: str | Unset = UNSET
        if not isinstance(self.confirmed_at, Unset):
            confirmed_at = self.confirmed_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "attempts": attempts,
            "id": id,
            "lastError": last_error,
            "origin": origin,
            "status": status,
        })
        if confirmed_at is not UNSET:
            field_dict["confirmedAt"] = confirmed_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        attempts = d.pop("attempts")

        id = d.pop("id")

        last_error = d.pop("lastError")

        origin = PartnerRequestItemOrigin(d.pop("origin"))




        status = PartnerRequestItemStatus(d.pop("status"))




        _confirmed_at = d.pop("confirmedAt", UNSET)
        confirmed_at: datetime.datetime | Unset
        if isinstance(_confirmed_at,  Unset):
            confirmed_at = UNSET
        else:
            confirmed_at = datetime.datetime.fromisoformat(_confirmed_at)




        partner_request_item = cls(
            at=at,
            attempts=attempts,
            id=id,
            last_error=last_error,
            origin=origin,
            status=status,
            confirmed_at=confirmed_at,
        )

        return partner_request_item

