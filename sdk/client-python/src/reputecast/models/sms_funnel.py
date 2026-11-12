from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.sms_funnel_reasons import SmsFunnelReasons





T = TypeVar("T", bound="SmsFunnel")



@_attrs_define
class SmsFunnel:
    """ 
        Attributes:
            assumed (int):
            base (int):
            can_receive (int):
            no_consent (int):
            reasons (SmsFunnelReasons):
            shared (int):
            skipped (int):
            skipped_in (list[str]):
            suppressed (int):
            to_send (int):
     """

    assumed: int
    base: int
    can_receive: int
    no_consent: int
    reasons: SmsFunnelReasons
    shared: int
    skipped: int
    skipped_in: list[str]
    suppressed: int
    to_send: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.sms_funnel_reasons import SmsFunnelReasons # noqa: PLC0415
        assumed = self.assumed

        base = self.base

        can_receive = self.can_receive

        no_consent = self.no_consent

        reasons = self.reasons.to_dict()

        shared = self.shared

        skipped = self.skipped

        skipped_in = self.skipped_in



        suppressed = self.suppressed

        to_send = self.to_send


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "assumed": assumed,
            "base": base,
            "canReceive": can_receive,
            "noConsent": no_consent,
            "reasons": reasons,
            "shared": shared,
            "skipped": skipped,
            "skippedIn": skipped_in,
            "suppressed": suppressed,
            "toSend": to_send,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sms_funnel_reasons import SmsFunnelReasons # noqa: PLC0415
        d = dict(src_dict)
        assumed = d.pop("assumed")

        base = d.pop("base")

        can_receive = d.pop("canReceive")

        no_consent = d.pop("noConsent")

        reasons = SmsFunnelReasons.from_dict(d.pop("reasons"))




        shared = d.pop("shared")

        skipped = d.pop("skipped")

        skipped_in = cast(list[str], d.pop("skippedIn"))


        suppressed = d.pop("suppressed")

        to_send = d.pop("toSend")

        sms_funnel = cls(
            assumed=assumed,
            base=base,
            can_receive=can_receive,
            no_consent=no_consent,
            reasons=reasons,
            shared=shared,
            skipped=skipped,
            skipped_in=skipped_in,
            suppressed=suppressed,
            to_send=to_send,
        )

        return sms_funnel

