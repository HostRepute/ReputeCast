from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.verify_contact_input_service import VerifyContactInputService
from ..types import UNSET, Unset






T = TypeVar("T", bound="VerifyContactInput")



@_attrs_define
class VerifyContactInput:
    """ 
        Attributes:
            service (VerifyContactInputService | Unset):
     """

    service: VerifyContactInputService | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        service: str | Unset = UNSET
        if not isinstance(self.service, Unset):
            service = self.service.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if service is not UNSET:
            field_dict["service"] = service

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        _service = d.pop("service", UNSET)
        service: VerifyContactInputService | Unset
        if isinstance(_service,  Unset):
            service = UNSET
        else:
            service = VerifyContactInputService(_service)




        verify_contact_input = cls(
            service=service,
        )

        return verify_contact_input

