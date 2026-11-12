from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.rendering_connect_input_region import RenderingConnectInputRegion
from ..models.rendering_connect_input_service import RenderingConnectInputService
from ..types import UNSET, Unset






T = TypeVar("T", bound="RenderingConnectInput")



@_attrs_define
class RenderingConnectInput:
    """ 
        Attributes:
            api_key (str): Write-only: sealed at rest and never returned
            service (RenderingConnectInputService):
            region (RenderingConnectInputRegion | Unset): Mailgun Inspect: the account's region (us by default)
     """

    api_key: str
    service: RenderingConnectInputService
    region: RenderingConnectInputRegion | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        api_key = self.api_key

        service = self.service.value

        region: str | Unset = UNSET
        if not isinstance(self.region, Unset):
            region = self.region.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "apiKey": api_key,
            "service": service,
        })
        if region is not UNSET:
            field_dict["region"] = region

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        api_key = d.pop("apiKey")

        service = RenderingConnectInputService(d.pop("service"))




        _region = d.pop("region", UNSET)
        region: RenderingConnectInputRegion | Unset
        if isinstance(_region,  Unset):
            region = UNSET
        else:
            region = RenderingConnectInputRegion(_region)




        rendering_connect_input = cls(
            api_key=api_key,
            service=service,
            region=region,
        )

        return rendering_connect_input

