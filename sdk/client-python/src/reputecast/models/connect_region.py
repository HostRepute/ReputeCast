from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="ConnectRegion")



@_attrs_define
class ConnectRegion:
    """ 
        Attributes:
            label (str):
            max24h (int):
            max_rate (float):
            name (str):
            production (bool):
            region (str):
            err (str | Unset):
     """

    label: str
    max24h: int
    max_rate: float
    name: str
    production: bool
    region: str
    err: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        label = self.label

        max24h = self.max24h

        max_rate = self.max_rate

        name = self.name

        production = self.production

        region = self.region

        err = self.err


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "label": label,
            "max24h": max24h,
            "maxRate": max_rate,
            "name": name,
            "production": production,
            "region": region,
        })
        if err is not UNSET:
            field_dict["err"] = err

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        label = d.pop("label")

        max24h = d.pop("max24h")

        max_rate = d.pop("maxRate")

        name = d.pop("name")

        production = d.pop("production")

        region = d.pop("region")

        err = d.pop("err", UNSET)

        connect_region = cls(
            label=label,
            max24h=max24h,
            max_rate=max_rate,
            name=name,
            production=production,
            region=region,
            err=err,
        )

        return connect_region

