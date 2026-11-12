from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.host_repute_asset_role import HostReputeAssetRole
from ..models.host_repute_asset_status import HostReputeAssetStatus
from ..models.host_repute_asset_type import HostReputeAssetType
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="HostReputeAsset")



@_attrs_define
class HostReputeAsset:
    """ 
        Attributes:
            asset (str):
            listings (list[str]):
            monitored (bool):
            role (HostReputeAssetRole):
            sources (int):
            type_ (HostReputeAssetType):
            dmarc_aligned (float | Unset): Percent of DMARC-aligned mail in the last 7 days
            dmarc_policy (str | Unset):
            last_sync_at (datetime.datetime | Unset):
            provider (str | Unset):
            score (int | Unset):
            status (HostReputeAssetStatus | Unset):
            verdict (str | Unset):
     """

    asset: str
    listings: list[str]
    monitored: bool
    role: HostReputeAssetRole
    sources: int
    type_: HostReputeAssetType
    dmarc_aligned: float | Unset = UNSET
    dmarc_policy: str | Unset = UNSET
    last_sync_at: datetime.datetime | Unset = UNSET
    provider: str | Unset = UNSET
    score: int | Unset = UNSET
    status: HostReputeAssetStatus | Unset = UNSET
    verdict: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        asset = self.asset

        listings = self.listings



        monitored = self.monitored

        role = self.role.value

        sources = self.sources

        type_ = self.type_.value

        dmarc_aligned = self.dmarc_aligned

        dmarc_policy = self.dmarc_policy

        last_sync_at: str | Unset = UNSET
        if not isinstance(self.last_sync_at, Unset):
            last_sync_at = self.last_sync_at.isoformat()

        provider = self.provider

        score = self.score

        status: str | Unset = UNSET
        if not isinstance(self.status, Unset):
            status = self.status.value


        verdict = self.verdict


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "asset": asset,
            "listings": listings,
            "monitored": monitored,
            "role": role,
            "sources": sources,
            "type": type_,
        })
        if dmarc_aligned is not UNSET:
            field_dict["dmarcAligned"] = dmarc_aligned
        if dmarc_policy is not UNSET:
            field_dict["dmarcPolicy"] = dmarc_policy
        if last_sync_at is not UNSET:
            field_dict["lastSyncAt"] = last_sync_at
        if provider is not UNSET:
            field_dict["provider"] = provider
        if score is not UNSET:
            field_dict["score"] = score
        if status is not UNSET:
            field_dict["status"] = status
        if verdict is not UNSET:
            field_dict["verdict"] = verdict

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        asset = d.pop("asset")

        listings = cast(list[str], d.pop("listings"))


        monitored = d.pop("monitored")

        role = HostReputeAssetRole(d.pop("role"))




        sources = d.pop("sources")

        type_ = HostReputeAssetType(d.pop("type"))




        dmarc_aligned = d.pop("dmarcAligned", UNSET)

        dmarc_policy = d.pop("dmarcPolicy", UNSET)

        _last_sync_at = d.pop("lastSyncAt", UNSET)
        last_sync_at: datetime.datetime | Unset
        if isinstance(_last_sync_at,  Unset):
            last_sync_at = UNSET
        else:
            last_sync_at = datetime.datetime.fromisoformat(_last_sync_at)




        provider = d.pop("provider", UNSET)

        score = d.pop("score", UNSET)

        _status = d.pop("status", UNSET)
        status: HostReputeAssetStatus | Unset
        if isinstance(_status,  Unset):
            status = UNSET
        else:
            status = HostReputeAssetStatus(_status)




        verdict = d.pop("verdict", UNSET)

        host_repute_asset = cls(
            asset=asset,
            listings=listings,
            monitored=monitored,
            role=role,
            sources=sources,
            type_=type_,
            dmarc_aligned=dmarc_aligned,
            dmarc_policy=dmarc_policy,
            last_sync_at=last_sync_at,
            provider=provider,
            score=score,
            status=status,
            verdict=verdict,
        )

        return host_repute_asset

