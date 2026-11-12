from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.suppression_provider_sync_status import SuppressionProviderSyncStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SuppressionProviderSync")



@_attrs_define
class SuppressionProviderSync:
    """ 
        Attributes:
            adapter (str):
            detail (str):
            local (bool):
            name (str):
            provider_id (str):
            pulled (int):
            pulled_note (str):
            pushed (int):
            status (SuppressionProviderSyncStatus):
            at (datetime.datetime | Unset):
     """

    adapter: str
    detail: str
    local: bool
    name: str
    provider_id: str
    pulled: int
    pulled_note: str
    pushed: int
    status: SuppressionProviderSyncStatus
    at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        adapter = self.adapter

        detail = self.detail

        local = self.local

        name = self.name

        provider_id = self.provider_id

        pulled = self.pulled

        pulled_note = self.pulled_note

        pushed = self.pushed

        status = self.status.value

        at: str | Unset = UNSET
        if not isinstance(self.at, Unset):
            at = self.at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "detail": detail,
            "local": local,
            "name": name,
            "providerId": provider_id,
            "pulled": pulled,
            "pulledNote": pulled_note,
            "pushed": pushed,
            "status": status,
        })
        if at is not UNSET:
            field_dict["at"] = at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        adapter = d.pop("adapter")

        detail = d.pop("detail")

        local = d.pop("local")

        name = d.pop("name")

        provider_id = d.pop("providerId")

        pulled = d.pop("pulled")

        pulled_note = d.pop("pulledNote")

        pushed = d.pop("pushed")

        status = SuppressionProviderSyncStatus(d.pop("status"))




        _at = d.pop("at", UNSET)
        at: datetime.datetime | Unset
        if isinstance(_at,  Unset):
            at = UNSET
        else:
            at = datetime.datetime.fromisoformat(_at)




        suppression_provider_sync = cls(
            adapter=adapter,
            detail=detail,
            local=local,
            name=name,
            provider_id=provider_id,
            pulled=pulled,
            pulled_note=pulled_note,
            pushed=pushed,
            status=status,
            at=at,
        )

        return suppression_provider_sync

