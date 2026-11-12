from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.deliverability_blocklist_source_delisting import DeliverabilityBlocklistSourceDelisting
from ..models.deliverability_blocklist_source_impact import DeliverabilityBlocklistSourceImpact
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="DeliverabilityBlocklistSource")



@_attrs_define
class DeliverabilityBlocklistSource:
    """ 
        Attributes:
            assets (list[str]):
            impact (DeliverabilityBlocklistSourceImpact):
            listed (bool):
            listed_assets (list[str]):
            source (str):
            delist_url (str | Unset):
            delisting (DeliverabilityBlocklistSourceDelisting | Unset):
            delisting_at (datetime.datetime | Unset):
            detail (str | Unset):
            since (datetime.datetime | Unset):
     """

    assets: list[str]
    impact: DeliverabilityBlocklistSourceImpact
    listed: bool
    listed_assets: list[str]
    source: str
    delist_url: str | Unset = UNSET
    delisting: DeliverabilityBlocklistSourceDelisting | Unset = UNSET
    delisting_at: datetime.datetime | Unset = UNSET
    detail: str | Unset = UNSET
    since: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        assets = self.assets



        impact = self.impact.value

        listed = self.listed

        listed_assets = self.listed_assets



        source = self.source

        delist_url = self.delist_url

        delisting: str | Unset = UNSET
        if not isinstance(self.delisting, Unset):
            delisting = self.delisting.value


        delisting_at: str | Unset = UNSET
        if not isinstance(self.delisting_at, Unset):
            delisting_at = self.delisting_at.isoformat()

        detail = self.detail

        since: str | Unset = UNSET
        if not isinstance(self.since, Unset):
            since = self.since.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "assets": assets,
            "impact": impact,
            "listed": listed,
            "listedAssets": listed_assets,
            "source": source,
        })
        if delist_url is not UNSET:
            field_dict["delistUrl"] = delist_url
        if delisting is not UNSET:
            field_dict["delisting"] = delisting
        if delisting_at is not UNSET:
            field_dict["delistingAt"] = delisting_at
        if detail is not UNSET:
            field_dict["detail"] = detail
        if since is not UNSET:
            field_dict["since"] = since

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        assets = cast(list[str], d.pop("assets"))


        impact = DeliverabilityBlocklistSourceImpact(d.pop("impact"))




        listed = d.pop("listed")

        listed_assets = cast(list[str], d.pop("listedAssets"))


        source = d.pop("source")

        delist_url = d.pop("delistUrl", UNSET)

        _delisting = d.pop("delisting", UNSET)
        delisting: DeliverabilityBlocklistSourceDelisting | Unset
        if isinstance(_delisting,  Unset):
            delisting = UNSET
        else:
            delisting = DeliverabilityBlocklistSourceDelisting(_delisting)




        _delisting_at = d.pop("delistingAt", UNSET)
        delisting_at: datetime.datetime | Unset
        if isinstance(_delisting_at,  Unset):
            delisting_at = UNSET
        else:
            delisting_at = datetime.datetime.fromisoformat(_delisting_at)




        detail = d.pop("detail", UNSET)

        _since = d.pop("since", UNSET)
        since: datetime.datetime | Unset
        if isinstance(_since,  Unset):
            since = UNSET
        else:
            since = datetime.datetime.fromisoformat(_since)




        deliverability_blocklist_source = cls(
            assets=assets,
            impact=impact,
            listed=listed,
            listed_assets=listed_assets,
            source=source,
            delist_url=delist_url,
            delisting=delisting,
            delisting_at=delisting_at,
            detail=detail,
            since=since,
        )

        return deliverability_blocklist_source

