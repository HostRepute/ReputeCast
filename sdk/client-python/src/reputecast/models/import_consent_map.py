from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.import_consent_map_pending import ImportConsentMapPending
from ..models.import_consent_map_subscribed import ImportConsentMapSubscribed
from ..models.import_consent_map_transactional import ImportConsentMapTransactional






T = TypeVar("T", bound="ImportConsentMap")



@_attrs_define
class ImportConsentMap:
    """ 
        Attributes:
            pending (ImportConsentMapPending):
            subscribed (ImportConsentMapSubscribed):
            transactional (ImportConsentMapTransactional):
     """

    pending: ImportConsentMapPending
    subscribed: ImportConsentMapSubscribed
    transactional: ImportConsentMapTransactional





    def to_dict(self) -> dict[str, Any]:
        pending = self.pending.value

        subscribed = self.subscribed.value

        transactional = self.transactional.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "pending": pending,
            "subscribed": subscribed,
            "transactional": transactional,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        pending = ImportConsentMapPending(d.pop("pending"))




        subscribed = ImportConsentMapSubscribed(d.pop("subscribed"))




        transactional = ImportConsentMapTransactional(d.pop("transactional"))




        import_consent_map = cls(
            pending=pending,
            subscribed=subscribed,
            transactional=transactional,
        )

        return import_consent_map

