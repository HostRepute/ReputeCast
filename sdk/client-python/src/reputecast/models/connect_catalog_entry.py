from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.connect_catalog_entry_bulk_mark import ConnectCatalogEntryBulkMark
from ..models.connect_catalog_entry_kind import ConnectCatalogEntryKind
from ..models.connect_catalog_entry_terms_tone import ConnectCatalogEntryTermsTone
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.connect_chip import ConnectChip
  from ..models.connect_price_list import ConnectPriceList





T = TypeVar("T", bound="ConnectCatalogEntry")



@_attrs_define
class ConnectCatalogEntry:
    """ 
        Attributes:
            adapter (str):
            bulk (str):
            bulk_allowed (bool):
            bulk_mark (ConnectCatalogEntryBulkMark):
            bulk_rank (int):
            capability_max (int):
            capability_score (int):
            chips (list[ConnectChip]):
            connected (int):
            dedicated_ips (bool):
            kind (ConnectCatalogEntryKind):
            name (str):
            pricing (str):
            recommended (bool):
            setup (str):
            short (str):
            terms (str):
            terms_tone (ConnectCatalogEntryTermsTone):
            webhooks (bool):
            price_list (ConnectPriceList | Unset):
     """

    adapter: str
    bulk: str
    bulk_allowed: bool
    bulk_mark: ConnectCatalogEntryBulkMark
    bulk_rank: int
    capability_max: int
    capability_score: int
    chips: list[ConnectChip]
    connected: int
    dedicated_ips: bool
    kind: ConnectCatalogEntryKind
    name: str
    pricing: str
    recommended: bool
    setup: str
    short: str
    terms: str
    terms_tone: ConnectCatalogEntryTermsTone
    webhooks: bool
    price_list: ConnectPriceList | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.connect_chip import ConnectChip # noqa: PLC0415
        from ..models.connect_price_list import ConnectPriceList # noqa: PLC0415
        adapter = self.adapter

        bulk = self.bulk

        bulk_allowed = self.bulk_allowed

        bulk_mark = self.bulk_mark.value

        bulk_rank = self.bulk_rank

        capability_max = self.capability_max

        capability_score = self.capability_score

        chips = []
        for chips_item_data in self.chips:
            chips_item = chips_item_data.to_dict()
            chips.append(chips_item)



        connected = self.connected

        dedicated_ips = self.dedicated_ips

        kind = self.kind.value

        name = self.name

        pricing = self.pricing

        recommended = self.recommended

        setup = self.setup

        short = self.short

        terms = self.terms

        terms_tone = self.terms_tone.value

        webhooks = self.webhooks

        price_list: dict[str, Any] | Unset = UNSET
        if not isinstance(self.price_list, Unset):
            price_list = self.price_list.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "bulk": bulk,
            "bulkAllowed": bulk_allowed,
            "bulkMark": bulk_mark,
            "bulkRank": bulk_rank,
            "capabilityMax": capability_max,
            "capabilityScore": capability_score,
            "chips": chips,
            "connected": connected,
            "dedicatedIps": dedicated_ips,
            "kind": kind,
            "name": name,
            "pricing": pricing,
            "recommended": recommended,
            "setup": setup,
            "short": short,
            "terms": terms,
            "termsTone": terms_tone,
            "webhooks": webhooks,
        })
        if price_list is not UNSET:
            field_dict["priceList"] = price_list

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.connect_chip import ConnectChip # noqa: PLC0415
        from ..models.connect_price_list import ConnectPriceList # noqa: PLC0415
        d = dict(src_dict)
        adapter = d.pop("adapter")

        bulk = d.pop("bulk")

        bulk_allowed = d.pop("bulkAllowed")

        bulk_mark = ConnectCatalogEntryBulkMark(d.pop("bulkMark"))




        bulk_rank = d.pop("bulkRank")

        capability_max = d.pop("capabilityMax")

        capability_score = d.pop("capabilityScore")

        chips = []
        _chips = d.pop("chips")
        for chips_item_data in (_chips):
            chips_item = ConnectChip.from_dict(chips_item_data)



            chips.append(chips_item)


        connected = d.pop("connected")

        dedicated_ips = d.pop("dedicatedIps")

        kind = ConnectCatalogEntryKind(d.pop("kind"))




        name = d.pop("name")

        pricing = d.pop("pricing")

        recommended = d.pop("recommended")

        setup = d.pop("setup")

        short = d.pop("short")

        terms = d.pop("terms")

        terms_tone = ConnectCatalogEntryTermsTone(d.pop("termsTone"))




        webhooks = d.pop("webhooks")

        _price_list = d.pop("priceList", UNSET)
        price_list: ConnectPriceList | Unset
        if isinstance(_price_list,  Unset):
            price_list = UNSET
        else:
            price_list = ConnectPriceList.from_dict(_price_list)




        connect_catalog_entry = cls(
            adapter=adapter,
            bulk=bulk,
            bulk_allowed=bulk_allowed,
            bulk_mark=bulk_mark,
            bulk_rank=bulk_rank,
            capability_max=capability_max,
            capability_score=capability_score,
            chips=chips,
            connected=connected,
            dedicated_ips=dedicated_ips,
            kind=kind,
            name=name,
            pricing=pricing,
            recommended=recommended,
            setup=setup,
            short=short,
            terms=terms,
            terms_tone=terms_tone,
            webhooks=webhooks,
            price_list=price_list,
        )

        return connect_catalog_entry

