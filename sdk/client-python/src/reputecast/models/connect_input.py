from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.connect_input_values import ConnectInputValues
  from ..models.connect_ses_input import ConnectSESInput
  from ..models.connect_smtp_input import ConnectSMTPInput





T = TypeVar("T", bound="ConnectInput")



@_attrs_define
class ConnectInput:
    """ 
        Attributes:
            public_url (str | Unset):
            ses (ConnectSESInput | Unset):
            smtp (ConnectSMTPInput | Unset):
            values (ConnectInputValues | Unset):
     """

    public_url: str | Unset = UNSET
    ses: ConnectSESInput | Unset = UNSET
    smtp: ConnectSMTPInput | Unset = UNSET
    values: ConnectInputValues | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.connect_input_values import ConnectInputValues # noqa: PLC0415
        from ..models.connect_ses_input import ConnectSESInput # noqa: PLC0415
        from ..models.connect_smtp_input import ConnectSMTPInput # noqa: PLC0415
        public_url = self.public_url

        ses: dict[str, Any] | Unset = UNSET
        if not isinstance(self.ses, Unset):
            ses = self.ses.to_dict()

        smtp: dict[str, Any] | Unset = UNSET
        if not isinstance(self.smtp, Unset):
            smtp = self.smtp.to_dict()

        values: dict[str, Any] | Unset = UNSET
        if not isinstance(self.values, Unset):
            values = self.values.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if public_url is not UNSET:
            field_dict["publicUrl"] = public_url
        if ses is not UNSET:
            field_dict["ses"] = ses
        if smtp is not UNSET:
            field_dict["smtp"] = smtp
        if values is not UNSET:
            field_dict["values"] = values

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.connect_input_values import ConnectInputValues # noqa: PLC0415
        from ..models.connect_ses_input import ConnectSESInput # noqa: PLC0415
        from ..models.connect_smtp_input import ConnectSMTPInput # noqa: PLC0415
        d = dict(src_dict)
        public_url = d.pop("publicUrl", UNSET)

        _ses = d.pop("ses", UNSET)
        ses: ConnectSESInput | Unset
        if isinstance(_ses,  Unset):
            ses = UNSET
        else:
            ses = ConnectSESInput.from_dict(_ses)




        _smtp = d.pop("smtp", UNSET)
        smtp: ConnectSMTPInput | Unset
        if isinstance(_smtp,  Unset):
            smtp = UNSET
        else:
            smtp = ConnectSMTPInput.from_dict(_smtp)




        _values = d.pop("values", UNSET)
        values: ConnectInputValues | Unset
        if isinstance(_values,  Unset):
            values = UNSET
        else:
            values = ConnectInputValues.from_dict(_values)




        connect_input = cls(
            public_url=public_url,
            ses=ses,
            smtp=smtp,
            values=values,
        )

        return connect_input

