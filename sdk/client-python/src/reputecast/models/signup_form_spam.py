from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SignupFormSpam")



@_attrs_define
class SignupFormSpam:
    """ 
        Attributes:
            block_disposable (bool):
            honeypot (bool):
            min_seconds (int): Submissions faster than this look automated
            rate_limit_per_ip (int): Submissions per minute from one address
            turnstile (bool):
     """

    block_disposable: bool
    honeypot: bool
    min_seconds: int
    rate_limit_per_ip: int
    turnstile: bool





    def to_dict(self) -> dict[str, Any]:
        block_disposable = self.block_disposable

        honeypot = self.honeypot

        min_seconds = self.min_seconds

        rate_limit_per_ip = self.rate_limit_per_ip

        turnstile = self.turnstile


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "blockDisposable": block_disposable,
            "honeypot": honeypot,
            "minSeconds": min_seconds,
            "rateLimitPerIp": rate_limit_per_ip,
            "turnstile": turnstile,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        block_disposable = d.pop("blockDisposable")

        honeypot = d.pop("honeypot")

        min_seconds = d.pop("minSeconds")

        rate_limit_per_ip = d.pop("rateLimitPerIp")

        turnstile = d.pop("turnstile")

        signup_form_spam = cls(
            block_disposable=block_disposable,
            honeypot=honeypot,
            min_seconds=min_seconds,
            rate_limit_per_ip=rate_limit_per_ip,
            turnstile=turnstile,
        )

        return signup_form_spam

