from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="DeliverabilityReputationListing")



@_attrs_define
class DeliverabilityReputationListing:
    """ 
        Attributes:
            asset (str):
            impact (str):
            source (str):
            delist_url (str | Unset):
            since (datetime.datetime | Unset):
     """

    asset: str
    impact: str
    source: str
    delist_url: str | Unset = UNSET
    since: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        asset = self.asset

        impact = self.impact

        source = self.source

        delist_url = self.delist_url

        since: str | Unset = UNSET
        if not isinstance(self.since, Unset):
            since = self.since.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "asset": asset,
            "impact": impact,
            "source": source,
        })
        if delist_url is not UNSET:
            field_dict["delistUrl"] = delist_url
        if since is not UNSET:
            field_dict["since"] = since

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        asset = d.pop("asset")

        impact = d.pop("impact")

        source = d.pop("source")

        delist_url = d.pop("delistUrl", UNSET)

        _since = d.pop("since", UNSET)
        since: datetime.datetime | Unset
        if isinstance(_since,  Unset):
            since = UNSET
        else:
            since = datetime.datetime.fromisoformat(_since)




        deliverability_reputation_listing = cls(
            asset=asset,
            impact=impact,
            source=source,
            delist_url=delist_url,
            since=since,
        )

        return deliverability_reputation_listing

