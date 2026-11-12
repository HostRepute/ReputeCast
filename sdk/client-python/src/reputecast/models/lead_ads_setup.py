from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.lead_ads_connection_item import LeadAdsConnectionItem





T = TypeVar("T", bound="LeadAdsSetup")



@_attrs_define
class LeadAdsSetup:
    """ 
        Attributes:
            permissions (list[str]):
            reachable (bool): Meta can reach it, so leads arrive within seconds; otherwise they're polled every 15 minutes
            url (str): The instance's public URL
            connection (LeadAdsConnectionItem | Unset):
     """

    permissions: list[str]
    reachable: bool
    url: str
    connection: LeadAdsConnectionItem | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.lead_ads_connection_item import LeadAdsConnectionItem # noqa: PLC0415
        permissions = self.permissions



        reachable = self.reachable

        url = self.url

        connection: dict[str, Any] | Unset = UNSET
        if not isinstance(self.connection, Unset):
            connection = self.connection.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "permissions": permissions,
            "reachable": reachable,
            "url": url,
        })
        if connection is not UNSET:
            field_dict["connection"] = connection

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.lead_ads_connection_item import LeadAdsConnectionItem # noqa: PLC0415
        d = dict(src_dict)
        permissions = cast(list[str], d.pop("permissions"))


        reachable = d.pop("reachable")

        url = d.pop("url")

        _connection = d.pop("connection", UNSET)
        connection: LeadAdsConnectionItem | Unset
        if isinstance(_connection,  Unset):
            connection = UNSET
        else:
            connection = LeadAdsConnectionItem.from_dict(_connection)




        lead_ads_setup = cls(
            permissions=permissions,
            reachable=reachable,
            url=url,
            connection=connection,
        )

        return lead_ads_setup

