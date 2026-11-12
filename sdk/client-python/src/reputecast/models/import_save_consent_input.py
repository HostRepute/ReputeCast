from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.import_save_consent_input_evidence import ImportSaveConsentInputEvidence
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.import_consent_map import ImportConsentMap





T = TypeVar("T", bound="ImportSaveConsentInput")



@_attrs_define
class ImportSaveConsentInput:
    """ 
        Attributes:
            evidence (ImportSaveConsentInputEvidence):
            map_ (ImportConsentMap):
            pull_suppressions (bool):
            record_history (bool):
            tag (str | Unset):
     """

    evidence: ImportSaveConsentInputEvidence
    map_: ImportConsentMap
    pull_suppressions: bool
    record_history: bool
    tag: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.import_consent_map import ImportConsentMap # noqa: PLC0415
        evidence = self.evidence.value

        map_ = self.map_.to_dict()

        pull_suppressions = self.pull_suppressions

        record_history = self.record_history

        tag = self.tag


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "evidence": evidence,
            "map": map_,
            "pullSuppressions": pull_suppressions,
            "recordHistory": record_history,
        })
        if tag is not UNSET:
            field_dict["tag"] = tag

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.import_consent_map import ImportConsentMap # noqa: PLC0415
        d = dict(src_dict)
        evidence = ImportSaveConsentInputEvidence(d.pop("evidence"))




        map_ = ImportConsentMap.from_dict(d.pop("map"))




        pull_suppressions = d.pop("pullSuppressions")

        record_history = d.pop("recordHistory")

        tag = d.pop("tag", UNSET)

        import_save_consent_input = cls(
            evidence=evidence,
            map_=map_,
            pull_suppressions=pull_suppressions,
            record_history=record_history,
            tag=tag,
        )

        return import_save_consent_input

