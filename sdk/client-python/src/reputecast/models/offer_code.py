from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.offer_code_status import OfferCodeStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="OfferCode")



@_attrs_define
class OfferCode:
    """ 
        Attributes:
            code (str):
            expires_at (datetime.datetime):
            issued_at (datetime.datetime):
            issued_to (str):
            status (OfferCodeStatus):
            redeemed_at (datetime.datetime | Unset):
     """

    code: str
    expires_at: datetime.datetime
    issued_at: datetime.datetime
    issued_to: str
    status: OfferCodeStatus
    redeemed_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        code = self.code

        expires_at = self.expires_at.isoformat()

        issued_at = self.issued_at.isoformat()

        issued_to = self.issued_to

        status = self.status.value

        redeemed_at: str | Unset = UNSET
        if not isinstance(self.redeemed_at, Unset):
            redeemed_at = self.redeemed_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "code": code,
            "expiresAt": expires_at,
            "issuedAt": issued_at,
            "issuedTo": issued_to,
            "status": status,
        })
        if redeemed_at is not UNSET:
            field_dict["redeemedAt"] = redeemed_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        code = d.pop("code")

        expires_at = datetime.datetime.fromisoformat(d.pop("expiresAt"))




        issued_at = datetime.datetime.fromisoformat(d.pop("issuedAt"))




        issued_to = d.pop("issuedTo")

        status = OfferCodeStatus(d.pop("status"))




        _redeemed_at = d.pop("redeemedAt", UNSET)
        redeemed_at: datetime.datetime | Unset
        if isinstance(_redeemed_at,  Unset):
            redeemed_at = UNSET
        else:
            redeemed_at = datetime.datetime.fromisoformat(_redeemed_at)




        offer_code = cls(
            code=code,
            expires_at=expires_at,
            issued_at=issued_at,
            issued_to=issued_to,
            status=status,
            redeemed_at=redeemed_at,
        )

        return offer_code

