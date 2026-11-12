from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.provider_quota_use_kind import ProviderQuotaUseKind






T = TypeVar("T", bound="ProviderQuotaUse")



@_attrs_define
class ProviderQuotaUse:
    """ 
        Attributes:
            from_failover (int):
            kind (ProviderQuotaUseKind):
            max_ (int):
            used (int):
     """

    from_failover: int
    kind: ProviderQuotaUseKind
    max_: int
    used: int





    def to_dict(self) -> dict[str, Any]:
        from_failover = self.from_failover

        kind = self.kind.value

        max_ = self.max_

        used = self.used


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "fromFailover": from_failover,
            "kind": kind,
            "max": max_,
            "used": used,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        from_failover = d.pop("fromFailover")

        kind = ProviderQuotaUseKind(d.pop("kind"))




        max_ = d.pop("max")

        used = d.pop("used")

        provider_quota_use = cls(
            from_failover=from_failover,
            kind=kind,
            max_=max_,
            used=used,
        )

        return provider_quota_use

