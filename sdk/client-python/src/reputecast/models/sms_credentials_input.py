from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.sms_credentials_input_fields import SmsCredentialsInputFields





T = TypeVar("T", bound="SmsCredentialsInput")



@_attrs_define
class SmsCredentialsInput:
    """ 
        Attributes:
            fields (SmsCredentialsInputFields):
     """

    fields: SmsCredentialsInputFields





    def to_dict(self) -> dict[str, Any]:
        from ..models.sms_credentials_input_fields import SmsCredentialsInputFields # noqa: PLC0415
        fields = self.fields.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "fields": fields,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sms_credentials_input_fields import SmsCredentialsInputFields # noqa: PLC0415
        d = dict(src_dict)
        fields = SmsCredentialsInputFields.from_dict(d.pop("fields"))




        sms_credentials_input = cls(
            fields=fields,
        )

        return sms_credentials_input

