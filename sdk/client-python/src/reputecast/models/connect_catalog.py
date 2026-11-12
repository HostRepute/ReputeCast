from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.connect_catalog_entry import ConnectCatalogEntry
  from ..models.connect_community_entry import ConnectCommunityEntry
  from ..models.connect_open_setup import ConnectOpenSetup





T = TypeVar("T", bound="ConnectCatalog")



@_attrs_define
class ConnectCatalog:
    """ 
        Attributes:
            builtin (list[ConnectCatalogEntry]):
            can_connect (bool):
            community (list[ConnectCommunityEntry]):
            helpers (list[str]):
            open_ (list[ConnectOpenSetup]):
     """

    builtin: list[ConnectCatalogEntry]
    can_connect: bool
    community: list[ConnectCommunityEntry]
    helpers: list[str]
    open_: list[ConnectOpenSetup]





    def to_dict(self) -> dict[str, Any]:
        from ..models.connect_catalog_entry import ConnectCatalogEntry # noqa: PLC0415
        from ..models.connect_community_entry import ConnectCommunityEntry # noqa: PLC0415
        from ..models.connect_open_setup import ConnectOpenSetup # noqa: PLC0415
        builtin = []
        for builtin_item_data in self.builtin:
            builtin_item = builtin_item_data.to_dict()
            builtin.append(builtin_item)



        can_connect = self.can_connect

        community = []
        for community_item_data in self.community:
            community_item = community_item_data.to_dict()
            community.append(community_item)



        helpers = self.helpers



        open_ = []
        for open_item_data in self.open_:
            open_item = open_item_data.to_dict()
            open_.append(open_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "builtin": builtin,
            "canConnect": can_connect,
            "community": community,
            "helpers": helpers,
            "open": open_,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.connect_catalog_entry import ConnectCatalogEntry # noqa: PLC0415
        from ..models.connect_community_entry import ConnectCommunityEntry # noqa: PLC0415
        from ..models.connect_open_setup import ConnectOpenSetup # noqa: PLC0415
        d = dict(src_dict)
        builtin = []
        _builtin = d.pop("builtin")
        for builtin_item_data in (_builtin):
            builtin_item = ConnectCatalogEntry.from_dict(builtin_item_data)



            builtin.append(builtin_item)


        can_connect = d.pop("canConnect")

        community = []
        _community = d.pop("community")
        for community_item_data in (_community):
            community_item = ConnectCommunityEntry.from_dict(community_item_data)



            community.append(community_item)


        helpers = cast(list[str], d.pop("helpers"))


        open_ = []
        _open_ = d.pop("open")
        for open_item_data in (_open_):
            open_item = ConnectOpenSetup.from_dict(open_item_data)



            open_.append(open_item)


        connect_catalog = cls(
            builtin=builtin,
            can_connect=can_connect,
            community=community,
            helpers=helpers,
            open_=open_,
        )

        return connect_catalog

