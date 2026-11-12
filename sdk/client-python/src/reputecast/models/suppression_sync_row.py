from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.suppression_sync_row_direction import SuppressionSyncRowDirection
from ..models.suppression_sync_row_status import SuppressionSyncRowStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SuppressionSyncRow")



@_attrs_define
class SuppressionSyncRow:
    """ 
        Attributes:
            direction (SuppressionSyncRowDirection):
            pending (int):
            provider (str):
            pulled (int):
            read_only (bool):
            scope (str):
            status (SuppressionSyncRowStatus):
            detail (str | Unset):
            last_sync (datetime.datetime | Unset):
            provider_id (str | Unset):
            pushed (int | Unset):
            tenant (str | Unset):
     """

    direction: SuppressionSyncRowDirection
    pending: int
    provider: str
    pulled: int
    read_only: bool
    scope: str
    status: SuppressionSyncRowStatus
    detail: str | Unset = UNSET
    last_sync: datetime.datetime | Unset = UNSET
    provider_id: str | Unset = UNSET
    pushed: int | Unset = UNSET
    tenant: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        direction = self.direction.value

        pending = self.pending

        provider = self.provider

        pulled = self.pulled

        read_only = self.read_only

        scope = self.scope

        status = self.status.value

        detail = self.detail

        last_sync: str | Unset = UNSET
        if not isinstance(self.last_sync, Unset):
            last_sync = self.last_sync.isoformat()

        provider_id = self.provider_id

        pushed = self.pushed

        tenant = self.tenant


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "direction": direction,
            "pending": pending,
            "provider": provider,
            "pulled": pulled,
            "readOnly": read_only,
            "scope": scope,
            "status": status,
        })
        if detail is not UNSET:
            field_dict["detail"] = detail
        if last_sync is not UNSET:
            field_dict["lastSync"] = last_sync
        if provider_id is not UNSET:
            field_dict["providerId"] = provider_id
        if pushed is not UNSET:
            field_dict["pushed"] = pushed
        if tenant is not UNSET:
            field_dict["tenant"] = tenant

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        direction = SuppressionSyncRowDirection(d.pop("direction"))




        pending = d.pop("pending")

        provider = d.pop("provider")

        pulled = d.pop("pulled")

        read_only = d.pop("readOnly")

        scope = d.pop("scope")

        status = SuppressionSyncRowStatus(d.pop("status"))




        detail = d.pop("detail", UNSET)

        _last_sync = d.pop("lastSync", UNSET)
        last_sync: datetime.datetime | Unset
        if isinstance(_last_sync,  Unset):
            last_sync = UNSET
        else:
            last_sync = datetime.datetime.fromisoformat(_last_sync)




        provider_id = d.pop("providerId", UNSET)

        pushed = d.pop("pushed", UNSET)

        tenant = d.pop("tenant", UNSET)

        suppression_sync_row = cls(
            direction=direction,
            pending=pending,
            provider=provider,
            pulled=pulled,
            read_only=read_only,
            scope=scope,
            status=status,
            detail=detail,
            last_sync=last_sync,
            provider_id=provider_id,
            pushed=pushed,
            tenant=tenant,
        )

        return suppression_sync_row

