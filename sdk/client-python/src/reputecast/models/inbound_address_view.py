from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="InboundAddressView")



@_attrs_define
class InboundAddressView:
    """ 
        Attributes:
            dmarc (str):
            fbl (str):
     """

    dmarc: str
    fbl: str





    def to_dict(self) -> dict[str, Any]:
        dmarc = self.dmarc

        fbl = self.fbl


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "dmarc": dmarc,
            "fbl": fbl,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        dmarc = d.pop("dmarc")

        fbl = d.pop("fbl")

        inbound_address_view = cls(
            dmarc=dmarc,
            fbl=fbl,
        )

        return inbound_address_view

