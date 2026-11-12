from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="SmsRegistrationHelper")



@_attrs_define
class SmsRegistrationHelper:
    """ 
        Attributes:
            help_reply (str):
            opt_in (str):
            opt_out_reply (str):
            samples (list[str]):
            terms_url (str):
     """

    help_reply: str
    opt_in: str
    opt_out_reply: str
    samples: list[str]
    terms_url: str





    def to_dict(self) -> dict[str, Any]:
        help_reply = self.help_reply

        opt_in = self.opt_in

        opt_out_reply = self.opt_out_reply

        samples = self.samples



        terms_url = self.terms_url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "helpReply": help_reply,
            "optIn": opt_in,
            "optOutReply": opt_out_reply,
            "samples": samples,
            "termsUrl": terms_url,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        help_reply = d.pop("helpReply")

        opt_in = d.pop("optIn")

        opt_out_reply = d.pop("optOutReply")

        samples = cast(list[str], d.pop("samples"))


        terms_url = d.pop("termsUrl")

        sms_registration_helper = cls(
            help_reply=help_reply,
            opt_in=opt_in,
            opt_out_reply=opt_out_reply,
            samples=samples,
            terms_url=terms_url,
        )

        return sms_registration_helper

