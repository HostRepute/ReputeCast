from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="SmsTermsContent")



@_attrs_define
class SmsTermsContent:
    """ 
        Attributes:
            carriers (str):
            cost (str):
            frequency (str):
            help_ (str):
            intro (str):
            opt_out (str):
            privacy (str):
            program (str):
            senders (str):
            address (str | Unset):
            help_email (str | Unset):
            help_number (str | Unset):
            privacy_url (str | Unset):
     """

    carriers: str
    cost: str
    frequency: str
    help_: str
    intro: str
    opt_out: str
    privacy: str
    program: str
    senders: str
    address: str | Unset = UNSET
    help_email: str | Unset = UNSET
    help_number: str | Unset = UNSET
    privacy_url: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        carriers = self.carriers

        cost = self.cost

        frequency = self.frequency

        help_ = self.help_

        intro = self.intro

        opt_out = self.opt_out

        privacy = self.privacy

        program = self.program

        senders = self.senders

        address = self.address

        help_email = self.help_email

        help_number = self.help_number

        privacy_url = self.privacy_url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "carriers": carriers,
            "cost": cost,
            "frequency": frequency,
            "help": help_,
            "intro": intro,
            "optOut": opt_out,
            "privacy": privacy,
            "program": program,
            "senders": senders,
        })
        if address is not UNSET:
            field_dict["address"] = address
        if help_email is not UNSET:
            field_dict["helpEmail"] = help_email
        if help_number is not UNSET:
            field_dict["helpNumber"] = help_number
        if privacy_url is not UNSET:
            field_dict["privacyUrl"] = privacy_url

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        carriers = d.pop("carriers")

        cost = d.pop("cost")

        frequency = d.pop("frequency")

        help_ = d.pop("help")

        intro = d.pop("intro")

        opt_out = d.pop("optOut")

        privacy = d.pop("privacy")

        program = d.pop("program")

        senders = d.pop("senders")

        address = d.pop("address", UNSET)

        help_email = d.pop("helpEmail", UNSET)

        help_number = d.pop("helpNumber", UNSET)

        privacy_url = d.pop("privacyUrl", UNSET)

        sms_terms_content = cls(
            carriers=carriers,
            cost=cost,
            frequency=frequency,
            help_=help_,
            intro=intro,
            opt_out=opt_out,
            privacy=privacy,
            program=program,
            senders=senders,
            address=address,
            help_email=help_email,
            help_number=help_number,
            privacy_url=privacy_url,
        )

        return sms_terms_content

