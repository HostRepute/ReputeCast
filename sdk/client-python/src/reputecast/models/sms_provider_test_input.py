from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="SmsProviderTestInput")



@_attrs_define
class SmsProviderTestInput:
    """ 
        Attributes:
            phone (str | Unset):
            setup_id (str | Unset):
            test_number_id (str | Unset):
     """

    phone: str | Unset = UNSET
    setup_id: str | Unset = UNSET
    test_number_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        phone = self.phone

        setup_id = self.setup_id

        test_number_id = self.test_number_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if phone is not UNSET:
            field_dict["phone"] = phone
        if setup_id is not UNSET:
            field_dict["setupId"] = setup_id
        if test_number_id is not UNSET:
            field_dict["testNumberId"] = test_number_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        phone = d.pop("phone", UNSET)

        setup_id = d.pop("setupId", UNSET)

        test_number_id = d.pop("testNumberId", UNSET)

        sms_provider_test_input = cls(
            phone=phone,
            setup_id=setup_id,
            test_number_id=test_number_id,
        )

        return sms_provider_test_input

