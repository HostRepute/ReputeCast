from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.adapter_builtin import AdapterBuiltin
  from ..models.adapter_community import AdapterCommunity





T = TypeVar("T", bound="AdaptersOverview")



@_attrs_define
class AdaptersOverview:
    """ 
        Attributes:
            builtin (list[AdapterBuiltin]):
            can_install (bool):
            community (list[AdapterCommunity]):
            http_adapter (bool):
            registry_url (str):
            sdk_url (str):
            template_url (str):
            up_to_date (bool):
            version (str):
            latest (str | Unset):
     """

    builtin: list[AdapterBuiltin]
    can_install: bool
    community: list[AdapterCommunity]
    http_adapter: bool
    registry_url: str
    sdk_url: str
    template_url: str
    up_to_date: bool
    version: str
    latest: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.adapter_builtin import AdapterBuiltin # noqa: PLC0415
        from ..models.adapter_community import AdapterCommunity # noqa: PLC0415
        builtin = []
        for builtin_item_data in self.builtin:
            builtin_item = builtin_item_data.to_dict()
            builtin.append(builtin_item)



        can_install = self.can_install

        community = []
        for community_item_data in self.community:
            community_item = community_item_data.to_dict()
            community.append(community_item)



        http_adapter = self.http_adapter

        registry_url = self.registry_url

        sdk_url = self.sdk_url

        template_url = self.template_url

        up_to_date = self.up_to_date

        version = self.version

        latest = self.latest


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "builtin": builtin,
            "canInstall": can_install,
            "community": community,
            "httpAdapter": http_adapter,
            "registryUrl": registry_url,
            "sdkUrl": sdk_url,
            "templateUrl": template_url,
            "upToDate": up_to_date,
            "version": version,
        })
        if latest is not UNSET:
            field_dict["latest"] = latest

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.adapter_builtin import AdapterBuiltin # noqa: PLC0415
        from ..models.adapter_community import AdapterCommunity # noqa: PLC0415
        d = dict(src_dict)
        builtin = []
        _builtin = d.pop("builtin")
        for builtin_item_data in (_builtin):
            builtin_item = AdapterBuiltin.from_dict(builtin_item_data)



            builtin.append(builtin_item)


        can_install = d.pop("canInstall")

        community = []
        _community = d.pop("community")
        for community_item_data in (_community):
            community_item = AdapterCommunity.from_dict(community_item_data)



            community.append(community_item)


        http_adapter = d.pop("httpAdapter")

        registry_url = d.pop("registryUrl")

        sdk_url = d.pop("sdkUrl")

        template_url = d.pop("templateUrl")

        up_to_date = d.pop("upToDate")

        version = d.pop("version")

        latest = d.pop("latest", UNSET)

        adapters_overview = cls(
            builtin=builtin,
            can_install=can_install,
            community=community,
            http_adapter=http_adapter,
            registry_url=registry_url,
            sdk_url=sdk_url,
            template_url=template_url,
            up_to_date=up_to_date,
            version=version,
            latest=latest,
        )

        return adapters_overview

