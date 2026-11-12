from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="VerificationDisconnectResult")



@_attrs_define
class VerificationDisconnectResult:
    """ 
        Attributes:
            bulk (bool):
            forms (list[str]):
            service (str):
     """

    bulk: bool
    forms: list[str]
    service: str





    def to_dict(self) -> dict[str, Any]:
        bulk = self.bulk

        forms = self.forms



        service = self.service


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bulk": bulk,
            "forms": forms,
            "service": service,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        bulk = d.pop("bulk")

        forms = cast(list[str], d.pop("forms"))


        service = d.pop("service")

        verification_disconnect_result = cls(
            bulk=bulk,
            forms=forms,
            service=service,
        )

        return verification_disconnect_result

