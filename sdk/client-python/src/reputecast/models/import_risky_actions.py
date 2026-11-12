from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.import_risky_actions_abuse import ImportRiskyActionsAbuse
from ..models.import_risky_actions_catch_all import ImportRiskyActionsCatchAll
from ..models.import_risky_actions_disposable import ImportRiskyActionsDisposable
from ..models.import_risky_actions_role import ImportRiskyActionsRole
from ..models.import_risky_actions_stale import ImportRiskyActionsStale
from ..models.import_risky_actions_trap import ImportRiskyActionsTrap
from ..models.import_risky_actions_typo import ImportRiskyActionsTypo
from ..models.import_risky_actions_unknown import ImportRiskyActionsUnknown
from ..types import UNSET, Unset






T = TypeVar("T", bound="ImportRiskyActions")



@_attrs_define
class ImportRiskyActions:
    """ 
        Attributes:
            disposable (ImportRiskyActionsDisposable):
            role (ImportRiskyActionsRole):
            stale (ImportRiskyActionsStale):
            trap (ImportRiskyActionsTrap):
            typo (ImportRiskyActionsTypo):
            abuse (ImportRiskyActionsAbuse | Unset):
            catch_all (ImportRiskyActionsCatchAll | Unset):
            unknown (ImportRiskyActionsUnknown | Unset):
     """

    disposable: ImportRiskyActionsDisposable
    role: ImportRiskyActionsRole
    stale: ImportRiskyActionsStale
    trap: ImportRiskyActionsTrap
    typo: ImportRiskyActionsTypo
    abuse: ImportRiskyActionsAbuse | Unset = UNSET
    catch_all: ImportRiskyActionsCatchAll | Unset = UNSET
    unknown: ImportRiskyActionsUnknown | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        disposable = self.disposable.value

        role = self.role.value

        stale = self.stale.value

        trap = self.trap.value

        typo = self.typo.value

        abuse: str | Unset = UNSET
        if not isinstance(self.abuse, Unset):
            abuse = self.abuse.value


        catch_all: str | Unset = UNSET
        if not isinstance(self.catch_all, Unset):
            catch_all = self.catch_all.value


        unknown: str | Unset = UNSET
        if not isinstance(self.unknown, Unset):
            unknown = self.unknown.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "disposable": disposable,
            "role": role,
            "stale": stale,
            "trap": trap,
            "typo": typo,
        })
        if abuse is not UNSET:
            field_dict["abuse"] = abuse
        if catch_all is not UNSET:
            field_dict["catch_all"] = catch_all
        if unknown is not UNSET:
            field_dict["unknown"] = unknown

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        disposable = ImportRiskyActionsDisposable(d.pop("disposable"))




        role = ImportRiskyActionsRole(d.pop("role"))




        stale = ImportRiskyActionsStale(d.pop("stale"))




        trap = ImportRiskyActionsTrap(d.pop("trap"))




        typo = ImportRiskyActionsTypo(d.pop("typo"))




        _abuse = d.pop("abuse", UNSET)
        abuse: ImportRiskyActionsAbuse | Unset
        if isinstance(_abuse,  Unset):
            abuse = UNSET
        else:
            abuse = ImportRiskyActionsAbuse(_abuse)




        _catch_all = d.pop("catch_all", UNSET)
        catch_all: ImportRiskyActionsCatchAll | Unset
        if isinstance(_catch_all,  Unset):
            catch_all = UNSET
        else:
            catch_all = ImportRiskyActionsCatchAll(_catch_all)




        _unknown = d.pop("unknown", UNSET)
        unknown: ImportRiskyActionsUnknown | Unset
        if isinstance(_unknown,  Unset):
            unknown = UNSET
        else:
            unknown = ImportRiskyActionsUnknown(_unknown)




        import_risky_actions = cls(
            disposable=disposable,
            role=role,
            stale=stale,
            trap=trap,
            typo=typo,
            abuse=abuse,
            catch_all=catch_all,
            unknown=unknown,
        )

        return import_risky_actions

