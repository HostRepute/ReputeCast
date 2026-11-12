from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_bounce_kind import TransactionalBounceKind
from ..types import UNSET, Unset






T = TypeVar("T", bound="TransactionalBounce")



@_attrs_define
class TransactionalBounce:
    """ 
        Attributes:
            kind (TransactionalBounceKind):
            class_ (str | Unset): mailbox_unknown, mailbox_full, domain_invalid, policy_block, spam_block, reputation,
                temporary, other
            diagnostic (str | Unset):  Example: 452 4.2.2 mailbox full.
     """

    kind: TransactionalBounceKind
    class_: str | Unset = UNSET
    diagnostic: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind.value

        class_ = self.class_

        diagnostic = self.diagnostic


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
        })
        if class_ is not UNSET:
            field_dict["class"] = class_
        if diagnostic is not UNSET:
            field_dict["diagnostic"] = diagnostic

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = TransactionalBounceKind(d.pop("kind"))




        class_ = d.pop("class", UNSET)

        diagnostic = d.pop("diagnostic", UNSET)

        transactional_bounce = cls(
            kind=kind,
            class_=class_,
            diagnostic=diagnostic,
        )

        return transactional_bounce

