from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.referral_network_kind import ReferralNetworkKind






T = TypeVar("T", bound="ReferralNetwork")



@_attrs_define
class ReferralNetwork:
    """ 
        Attributes:
            asn (str):
            cidr (str):
            kind (ReferralNetworkKind):
            name (str):
     """

    asn: str
    cidr: str
    kind: ReferralNetworkKind
    name: str





    def to_dict(self) -> dict[str, Any]:
        asn = self.asn

        cidr = self.cidr

        kind = self.kind.value

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "asn": asn,
            "cidr": cidr,
            "kind": kind,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        asn = d.pop("asn")

        cidr = d.pop("cidr")

        kind = ReferralNetworkKind(d.pop("kind"))




        name = d.pop("name")

        referral_network = cls(
            asn=asn,
            cidr=cidr,
            kind=kind,
            name=name,
        )

        return referral_network

