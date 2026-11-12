from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.host_repute_data_source_via import HostReputeDataSourceVia






T = TypeVar("T", bound="HostReputeDataSource")



@_attrs_define
class HostReputeDataSource:
    """ 
        Attributes:
            via (HostReputeDataSourceVia):
     """

    via: HostReputeDataSourceVia





    def to_dict(self) -> dict[str, Any]:
        via = self.via.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "via": via,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        via = HostReputeDataSourceVia(d.pop("via"))




        host_repute_data_source = cls(
            via=via,
        )

        return host_repute_data_source

