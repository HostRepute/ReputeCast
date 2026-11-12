from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.deliverability_blocklist_status_interval import DeliverabilityBlocklistStatusInterval
from ..models.deliverability_blocklist_status_via import DeliverabilityBlocklistStatusVia
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.deliverability_blocklist_source import DeliverabilityBlocklistSource





T = TypeVar("T", bound="DeliverabilityBlocklistStatus")



@_attrs_define
class DeliverabilityBlocklistStatus:
    """ 
        Attributes:
            assets (list[str]):
            clean (int):
            interval (DeliverabilityBlocklistStatusInterval):
            listed (int):
            results (list[DeliverabilityBlocklistSource]):
            sources (int):
            stale (bool):
            via (DeliverabilityBlocklistStatusVia):
            checked_at (datetime.datetime | Unset):
     """

    assets: list[str]
    clean: int
    interval: DeliverabilityBlocklistStatusInterval
    listed: int
    results: list[DeliverabilityBlocklistSource]
    sources: int
    stale: bool
    via: DeliverabilityBlocklistStatusVia
    checked_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.deliverability_blocklist_source import DeliverabilityBlocklistSource # noqa: PLC0415
        assets = self.assets



        clean = self.clean

        interval = self.interval.value

        listed = self.listed

        results = []
        for results_item_data in self.results:
            results_item = results_item_data.to_dict()
            results.append(results_item)



        sources = self.sources

        stale = self.stale

        via = self.via.value

        checked_at: str | Unset = UNSET
        if not isinstance(self.checked_at, Unset):
            checked_at = self.checked_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "assets": assets,
            "clean": clean,
            "interval": interval,
            "listed": listed,
            "results": results,
            "sources": sources,
            "stale": stale,
            "via": via,
        })
        if checked_at is not UNSET:
            field_dict["checkedAt"] = checked_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.deliverability_blocklist_source import DeliverabilityBlocklistSource # noqa: PLC0415
        d = dict(src_dict)
        assets = cast(list[str], d.pop("assets"))


        clean = d.pop("clean")

        interval = DeliverabilityBlocklistStatusInterval(d.pop("interval"))




        listed = d.pop("listed")

        results = []
        _results = d.pop("results")
        for results_item_data in (_results):
            results_item = DeliverabilityBlocklistSource.from_dict(results_item_data)



            results.append(results_item)


        sources = d.pop("sources")

        stale = d.pop("stale")

        via = DeliverabilityBlocklistStatusVia(d.pop("via"))




        _checked_at = d.pop("checkedAt", UNSET)
        checked_at: datetime.datetime | Unset
        if isinstance(_checked_at,  Unset):
            checked_at = UNSET
        else:
            checked_at = datetime.datetime.fromisoformat(_checked_at)




        deliverability_blocklist_status = cls(
            assets=assets,
            clean=clean,
            interval=interval,
            listed=listed,
            results=results,
            sources=sources,
            stale=stale,
            via=via,
            checked_at=checked_at,
        )

        return deliverability_blocklist_status

