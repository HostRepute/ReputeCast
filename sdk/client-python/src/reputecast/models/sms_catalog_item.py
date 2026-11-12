from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_catalog_item_webhooks import SmsCatalogItemWebhooks
from typing import cast

if TYPE_CHECKING:
  from ..models.badge import Badge
  from ..models.field import Field





T = TypeVar("T", bound="SmsCatalogItem")



@_attrs_define
class SmsCatalogItem:
    """ 
        Attributes:
            adapter (str):
            badges (list[Badge]):
            connected (int):
            fields (list[Field]):
            name (str):
            summary (str):
            version (str):
            webhooks (SmsCatalogItemWebhooks):
     """

    adapter: str
    badges: list[Badge]
    connected: int
    fields: list[Field]
    name: str
    summary: str
    version: str
    webhooks: SmsCatalogItemWebhooks





    def to_dict(self) -> dict[str, Any]:
        from ..models.badge import Badge # noqa: PLC0415
        from ..models.field import Field # noqa: PLC0415
        adapter = self.adapter

        badges = []
        for badges_item_data in self.badges:
            badges_item = badges_item_data.to_dict()
            badges.append(badges_item)



        connected = self.connected

        fields = []
        for fields_item_data in self.fields:
            fields_item = fields_item_data.to_dict()
            fields.append(fields_item)



        name = self.name

        summary = self.summary

        version = self.version

        webhooks = self.webhooks.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "badges": badges,
            "connected": connected,
            "fields": fields,
            "name": name,
            "summary": summary,
            "version": version,
            "webhooks": webhooks,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.badge import Badge # noqa: PLC0415
        from ..models.field import Field # noqa: PLC0415
        d = dict(src_dict)
        adapter = d.pop("adapter")

        badges = []
        _badges = d.pop("badges")
        for badges_item_data in (_badges):
            badges_item = Badge.from_dict(badges_item_data)



            badges.append(badges_item)


        connected = d.pop("connected")

        fields = []
        _fields = d.pop("fields")
        for fields_item_data in (_fields):
            fields_item = Field.from_dict(fields_item_data)



            fields.append(fields_item)


        name = d.pop("name")

        summary = d.pop("summary")

        version = d.pop("version")

        webhooks = SmsCatalogItemWebhooks(d.pop("webhooks"))




        sms_catalog_item = cls(
            adapter=adapter,
            badges=badges,
            connected=connected,
            fields=fields,
            name=name,
            summary=summary,
            version=version,
            webhooks=webhooks,
        )

        return sms_catalog_item

