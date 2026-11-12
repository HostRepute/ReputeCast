from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="PlacementAuthCheck")



@_attrs_define
class PlacementAuthCheck:
    """ 
        Attributes:
            result (str):
            aligned (bool | Unset):
            detail (str | Unset):
            domain (str | Unset):
            ip (str | Unset):
            selector (str | Unset):
     """

    result: str
    aligned: bool | Unset = UNSET
    detail: str | Unset = UNSET
    domain: str | Unset = UNSET
    ip: str | Unset = UNSET
    selector: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        result = self.result

        aligned = self.aligned

        detail = self.detail

        domain = self.domain

        ip = self.ip

        selector = self.selector


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "result": result,
        })
        if aligned is not UNSET:
            field_dict["aligned"] = aligned
        if detail is not UNSET:
            field_dict["detail"] = detail
        if domain is not UNSET:
            field_dict["domain"] = domain
        if ip is not UNSET:
            field_dict["ip"] = ip
        if selector is not UNSET:
            field_dict["selector"] = selector

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        result = d.pop("result")

        aligned = d.pop("aligned", UNSET)

        detail = d.pop("detail", UNSET)

        domain = d.pop("domain", UNSET)

        ip = d.pop("ip", UNSET)

        selector = d.pop("selector", UNSET)

        placement_auth_check = cls(
            result=result,
            aligned=aligned,
            detail=detail,
            domain=domain,
            ip=ip,
            selector=selector,
        )

        return placement_auth_check

