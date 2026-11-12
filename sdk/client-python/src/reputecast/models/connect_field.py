from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.connect_field_type import ConnectFieldType
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.connect_option import ConnectOption





T = TypeVar("T", bound="ConnectField")



@_attrs_define
class ConnectField:
    """ 
        Attributes:
            key (str):
            label (str):
            options (list[ConnectOption]):
            required (bool):
            secret (bool):
            set_ (bool):
            type_ (ConnectFieldType):
            value (str):
            help_ (str | Unset):
     """

    key: str
    label: str
    options: list[ConnectOption]
    required: bool
    secret: bool
    set_: bool
    type_: ConnectFieldType
    value: str
    help_: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.connect_option import ConnectOption # noqa: PLC0415
        key = self.key

        label = self.label

        options = []
        for options_item_data in self.options:
            options_item = options_item_data.to_dict()
            options.append(options_item)



        required = self.required

        secret = self.secret

        set_ = self.set_

        type_ = self.type_.value

        value = self.value

        help_ = self.help_


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "label": label,
            "options": options,
            "required": required,
            "secret": secret,
            "set": set_,
            "type": type_,
            "value": value,
        })
        if help_ is not UNSET:
            field_dict["help"] = help_

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.connect_option import ConnectOption # noqa: PLC0415
        d = dict(src_dict)
        key = d.pop("key")

        label = d.pop("label")

        options = []
        _options = d.pop("options")
        for options_item_data in (_options):
            options_item = ConnectOption.from_dict(options_item_data)



            options.append(options_item)


        required = d.pop("required")

        secret = d.pop("secret")

        set_ = d.pop("set")

        type_ = ConnectFieldType(d.pop("type"))




        value = d.pop("value")

        help_ = d.pop("help", UNSET)

        connect_field = cls(
            key=key,
            label=label,
            options=options,
            required=required,
            secret=secret,
            set_=set_,
            type_=type_,
            value=value,
            help_=help_,
        )

        return connect_field

