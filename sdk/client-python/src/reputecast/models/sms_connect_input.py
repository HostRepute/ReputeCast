from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.sms_connect_input_fields import SmsConnectInputFields





T = TypeVar("T", bound="SmsConnectInput")



@_attrs_define
class SmsConnectInput:
    """ 
        Attributes:
            adapter (str):
            fields (SmsConnectInputFields):
            name (str | Unset):
            setup_id (str | Unset):
     """

    adapter: str
    fields: SmsConnectInputFields
    name: str | Unset = UNSET
    setup_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sms_connect_input_fields import SmsConnectInputFields # noqa: PLC0415
        adapter = self.adapter

        fields = self.fields.to_dict()

        name = self.name

        setup_id = self.setup_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "fields": fields,
        })
        if name is not UNSET:
            field_dict["name"] = name
        if setup_id is not UNSET:
            field_dict["setupId"] = setup_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sms_connect_input_fields import SmsConnectInputFields # noqa: PLC0415
        d = dict(src_dict)
        adapter = d.pop("adapter")

        fields = SmsConnectInputFields.from_dict(d.pop("fields"))




        name = d.pop("name", UNSET)

        setup_id = d.pop("setupId", UNSET)

        sms_connect_input = cls(
            adapter=adapter,
            fields=fields,
            name=name,
            setup_id=setup_id,
        )

        return sms_connect_input

