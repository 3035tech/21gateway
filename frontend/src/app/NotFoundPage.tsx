import { ArrowLeft } from 'lucide-react'
import { Link } from 'react-router-dom'

export function NotFoundPage() {
  return (
    <section className="mx-auto max-w-xl py-12 text-center">
      <p className="text-6xl font-bold text-brand-purple" aria-hidden="true">
        404
      </p>
      <h1 className="mt-5 text-2xl font-bold">Página não encontrada</h1>
      <p className="mt-3 leading-7 text-slate-600">
        O endereço pode ter mudado ou a página ainda não está disponível.
      </p>
      <Link to="/" className="button-primary mt-7">
        <ArrowLeft size={16} aria-hidden="true" />
        Voltar à visão geral
      </Link>
    </section>
  )
}
