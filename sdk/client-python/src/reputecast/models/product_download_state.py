from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.product_download_state_revoke_reason import ProductDownloadStateRevokeReason
from ..models.product_download_state_state import ProductDownloadStateState
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ProductDownloadState")



@_attrs_define
class ProductDownloadState:
    """ 
        Attributes:
            allowed (int):
            new_link (bool):
            new_link_by (str):
            provider (str):
            revoke_reason (ProductDownloadStateRevokeReason):
            revoked_by (str):
            state (ProductDownloadStateState):
            used (int):
            delivered_at (datetime.datetime | Unset):
            error (str | Unset):
            expires_at (datetime.datetime | Unset):
            new_link_at (datetime.datetime | Unset):
            revoked_at (datetime.datetime | Unset):
            sent_at (datetime.datetime | Unset):
     """

    allowed: int
    new_link: bool
    new_link_by: str
    provider: str
    revoke_reason: ProductDownloadStateRevokeReason
    revoked_by: str
    state: ProductDownloadStateState
    used: int
    delivered_at: datetime.datetime | Unset = UNSET
    error: str | Unset = UNSET
    expires_at: datetime.datetime | Unset = UNSET
    new_link_at: datetime.datetime | Unset = UNSET
    revoked_at: datetime.datetime | Unset = UNSET
    sent_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        allowed = self.allowed

        new_link = self.new_link

        new_link_by = self.new_link_by

        provider = self.provider

        revoke_reason = self.revoke_reason.value

        revoked_by = self.revoked_by

        state = self.state.value

        used = self.used

        delivered_at: str | Unset = UNSET
        if not isinstance(self.delivered_at, Unset):
            delivered_at = self.delivered_at.isoformat()

        error = self.error

        expires_at: str | Unset = UNSET
        if not isinstance(self.expires_at, Unset):
            expires_at = self.expires_at.isoformat()

        new_link_at: str | Unset = UNSET
        if not isinstance(self.new_link_at, Unset):
            new_link_at = self.new_link_at.isoformat()

        revoked_at: str | Unset = UNSET
        if not isinstance(self.revoked_at, Unset):
            revoked_at = self.revoked_at.isoformat()

        sent_at: str | Unset = UNSET
        if not isinstance(self.sent_at, Unset):
            sent_at = self.sent_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "allowed": allowed,
            "newLink": new_link,
            "newLinkBy": new_link_by,
            "provider": provider,
            "revokeReason": revoke_reason,
            "revokedBy": revoked_by,
            "state": state,
            "used": used,
        })
        if delivered_at is not UNSET:
            field_dict["deliveredAt"] = delivered_at
        if error is not UNSET:
            field_dict["error"] = error
        if expires_at is not UNSET:
            field_dict["expiresAt"] = expires_at
        if new_link_at is not UNSET:
            field_dict["newLinkAt"] = new_link_at
        if revoked_at is not UNSET:
            field_dict["revokedAt"] = revoked_at
        if sent_at is not UNSET:
            field_dict["sentAt"] = sent_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        allowed = d.pop("allowed")

        new_link = d.pop("newLink")

        new_link_by = d.pop("newLinkBy")

        provider = d.pop("provider")

        revoke_reason = ProductDownloadStateRevokeReason(d.pop("revokeReason"))




        revoked_by = d.pop("revokedBy")

        state = ProductDownloadStateState(d.pop("state"))




        used = d.pop("used")

        _delivered_at = d.pop("deliveredAt", UNSET)
        delivered_at: datetime.datetime | Unset
        if isinstance(_delivered_at,  Unset):
            delivered_at = UNSET
        else:
            delivered_at = datetime.datetime.fromisoformat(_delivered_at)




        error = d.pop("error", UNSET)

        _expires_at = d.pop("expiresAt", UNSET)
        expires_at: datetime.datetime | Unset
        if isinstance(_expires_at,  Unset):
            expires_at = UNSET
        else:
            expires_at = datetime.datetime.fromisoformat(_expires_at)




        _new_link_at = d.pop("newLinkAt", UNSET)
        new_link_at: datetime.datetime | Unset
        if isinstance(_new_link_at,  Unset):
            new_link_at = UNSET
        else:
            new_link_at = datetime.datetime.fromisoformat(_new_link_at)




        _revoked_at = d.pop("revokedAt", UNSET)
        revoked_at: datetime.datetime | Unset
        if isinstance(_revoked_at,  Unset):
            revoked_at = UNSET
        else:
            revoked_at = datetime.datetime.fromisoformat(_revoked_at)




        _sent_at = d.pop("sentAt", UNSET)
        sent_at: datetime.datetime | Unset
        if isinstance(_sent_at,  Unset):
            sent_at = UNSET
        else:
            sent_at = datetime.datetime.fromisoformat(_sent_at)




        product_download_state = cls(
            allowed=allowed,
            new_link=new_link,
            new_link_by=new_link_by,
            provider=provider,
            revoke_reason=revoke_reason,
            revoked_by=revoked_by,
            state=state,
            used=used,
            delivered_at=delivered_at,
            error=error,
            expires_at=expires_at,
            new_link_at=new_link_at,
            revoked_at=revoked_at,
            sent_at=sent_at,
        )

        return product_download_state

